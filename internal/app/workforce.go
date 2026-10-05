package app

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/core"
)

func (s *Service) CreateMemoryItem(v core.MemoryItem) (core.MemoryItem,error) {
	v.Namespace=strings.TrimSpace(v.Namespace)
	v.Title=strings.TrimSpace(v.Title)
	v.Content=strings.TrimSpace(v.Content)
	if v.Namespace==""||v.Title==""||v.Content==""{
		return core.MemoryItem{},errors.New("namespace, title and content are required")
	}
	for _,agentID:=range v.AllowedAgentIDs{
		if _,err:=s.store.Agent(agentID);err!=nil{return core.MemoryItem{},errors.New("allowed agent not found: "+agentID)}
	}
	now:=s.now()
	v.ID=core.NewID("mem")
	v.CreatedAt=now
	v.UpdatedAt=now
	if err:=s.store.PutMemory(v);err!=nil{return core.MemoryItem{},err}
	_ = s.audit("", "memory.item_created","success","memory item "+v.Title+" created",map[string]any{"memory_id":v.ID})
	return v,nil
}

func (s *Service) SearchMemory(query,namespace,agentID string) []core.MemoryItem {
	query=strings.ToLower(strings.TrimSpace(query))
	namespace=strings.TrimSpace(namespace)
	out:=[]core.MemoryItem{}
	for _,item:=range s.store.MemoryItems(){
		if namespace!=""&&!strings.EqualFold(namespace,item.Namespace){continue}
		if agentID!=""&&len(item.AllowedAgentIDs)>0&&!containsFold(item.AllowedAgentIDs,agentID){continue}
		if query!=""{
			haystack:=strings.ToLower(item.Title+" "+item.Content+" "+strings.Join(item.Tags," "))
			if !strings.Contains(haystack,query){continue}
		}
		out=append(out,item)
		if len(out)>=100{break}
	}
	sort.Slice(out,func(i,j int)bool{return out[i].UpdatedAt.After(out[j].UpdatedAt)})
	return out
}

func (s *Service) AddCertification(v core.Certification) (core.Certification,error) {
	if _,err:=s.store.Agent(v.AgentID);err!=nil{return core.Certification{},err}
	v.Skill=strings.TrimSpace(v.Skill)
	v.Issuer=strings.TrimSpace(v.Issuer)
	if v.Skill==""||v.Issuer==""{return core.Certification{},errors.New("skill and issuer are required")}
	if v.Score<0||v.Score>100{return core.Certification{},errors.New("score must be between 0 and 100")}
	v.ID=core.NewID("cert")
	if v.IssuedAt.IsZero(){v.IssuedAt=s.now()}
	if err:=s.store.PutCertification(v);err!=nil{return core.Certification{},err}
	_ = s.audit(v.AgentID,"training.certification_issued","success","certified for "+v.Skill,map[string]any{"score":v.Score,"certification_id":v.ID})
	return v,nil
}

func (s *Service) Certifications(agentID string) []core.Certification {
	return s.store.Certifications(agentID)
}

func (s *Service) CreateFlow(v core.FlowDefinition) (core.FlowDefinition,error) {
	v.Name=strings.TrimSpace(v.Name)
	if v.Name==""||len(v.Steps)==0{return core.FlowDefinition{},errors.New("name and at least one step are required")}
	for i:=range v.Steps{
		if _,err:=s.store.Agent(v.Steps[i].AgentID);err!=nil{return core.FlowDefinition{},errors.New("step agent not found: "+v.Steps[i].AgentID)}
		if strings.TrimSpace(v.Steps[i].Tool)==""||strings.TrimSpace(v.Steps[i].Action)==""{return core.FlowDefinition{},errors.New("each step requires tool and action")}
		if v.Steps[i].ID==""{v.Steps[i].ID=core.NewID("step")}
	}
	now:=s.now()
	v.ID=core.NewID("flow")
	v.CreatedAt=now
	v.UpdatedAt=now
	if err:=s.store.PutFlow(v);err!=nil{return core.FlowDefinition{},err}
	return v,nil
}

func (s *Service) Flows() []core.FlowDefinition { return s.store.Flows() }

func (s *Service) StartFlow(flowID string) (core.FlowRun,error) {
	if _,err:=s.store.Flow(flowID);err!=nil{return core.FlowRun{},err}
	now:=s.now()
	run:=core.FlowRun{ID:core.NewID("run"),FlowID:flowID,Status:"running",History:[]core.FlowStepResult{},CreatedAt:now,UpdatedAt:now}
	if err:=s.store.PutFlowRun(run);err!=nil{return core.FlowRun{},err}
	return run,nil
}

func (s *Service) FlowRuns() []core.FlowRun { return s.store.FlowRuns() }

func (s *Service) CurrentFlowStep(runID string) (core.FlowRun,*core.FlowStep,*core.GateResult,error) {
	run,err:=s.store.FlowRun(runID)
	if err!=nil{return core.FlowRun{},nil,nil,err}
	flow,err:=s.store.Flow(run.FlowID)
	if err!=nil{return core.FlowRun{},nil,nil,err}
	if run.Status=="completed"||run.Status=="failed"||run.Current>=len(flow.Steps){return run,nil,nil,nil}
	step:=flow.Steps[run.Current]
	if run.PendingApprovalID!=""{
		approval,err:=s.store.Approval(run.PendingApprovalID)
		if err!=nil{return core.FlowRun{},nil,nil,err}
		if approval.Status==core.ApprovalPending{
			result:=core.GateResult{Decision:core.DecisionApproval,ApprovalID:approval.ID,PolicyID:approval.PolicyID,Reason:"waiting for human approval"}
			return run,&step,&result,nil
		}
		if approval.Status==core.ApprovalRejected{
			run.Status="failed"
			run.UpdatedAt=s.now()
			run.History=append(run.History,core.FlowStepResult{StepID:step.ID,Status:"rejected",Message:"human approval rejected",CompletedAt:s.now()})
			run.PendingApprovalID=""
			_ = s.store.PutFlowRun(run)
			return run,nil,nil,nil
		}
		run.PendingApprovalID=""
		run.Status="running"
		run.UpdatedAt=s.now()
		_ = s.store.PutFlowRun(run)
		result:=core.GateResult{Decision:core.DecisionAllow,PolicyID:approval.PolicyID,Reason:"human approval granted"}
		return run,&step,&result,nil
	}
	result,err:=s.Evaluate(core.ActionRequest{AgentID:step.AgentID,Tool:step.Tool,Action:step.Action,AmountUSD:step.AmountUSD,TraceID:run.ID})
	if err!=nil{return core.FlowRun{},nil,nil,err}
	if result.Decision==core.DecisionApproval{
		run.PendingApprovalID=result.ApprovalID
		run.Status="waiting_approval"
		run.UpdatedAt=s.now()
		_ = s.store.PutFlowRun(run)
	}
	if result.Decision==core.DecisionDeny{
		run.Status="failed"
		run.UpdatedAt=s.now()
		run.History=append(run.History,core.FlowStepResult{StepID:step.ID,Status:"denied",Message:result.Reason,CompletedAt:s.now()})
		_ = s.store.PutFlowRun(run)
	}
	return run,&step,&result,nil
}

func (s *Service) CompleteFlowStep(runID,status,message string) (core.FlowRun,error) {
	run,err:=s.store.FlowRun(runID)
	if err!=nil{return core.FlowRun{},err}
	flow,err:=s.store.Flow(run.FlowID)
	if err!=nil{return core.FlowRun{},err}
	if run.Status=="waiting_approval"{return core.FlowRun{},errors.New("flow is waiting for human approval")}
	if run.Status!="running"{return core.FlowRun{},errors.New("flow is not running")}
	if run.Current>=len(flow.Steps){return core.FlowRun{},errors.New("flow has no current step")}
	if status!="success"&&status!="failed"{return core.FlowRun{},errors.New("status must be success or failed")}
	step:=flow.Steps[run.Current]
	now:=s.now()
	run.History=append(run.History,core.FlowStepResult{StepID:step.ID,Status:status,Message:message,CompletedAt:now})
	if status=="failed"{
		run.Status="failed"
	}else{
		run.Current++
		if run.Current>=len(flow.Steps){run.Status="completed"}
	}
	run.UpdatedAt=now
	if err:=s.store.PutFlowRun(run);err!=nil{return core.FlowRun{},err}
	eventType:="task.completed"
	if status=="failed"{eventType="task.failed"}
	_,_ = s.RecordEvent(core.Event{AgentID:step.AgentID,Type:eventType,Status:status,TraceID:run.ID,Tool:step.Tool,Action:step.Action,Message:message,OccurredAt:now})
	return run,nil
}

func (s *Service) WorkforceHeadcount() map[string]any {
	agents:=s.store.Agents()
	byDepartment:=map[string]int{}
	byStatus:=map[string]int{}
	for _,a:=range agents{
		dept:=a.Department
		if dept==""{dept="Unassigned"}
		byDepartment[dept]++
		byStatus[string(a.Status)]++
	}
	return map[string]any{
		"total":len(agents),
		"by_department":byDepartment,
		"by_status":byStatus,
		"generated_at":time.Now().UTC(),
	}
}
