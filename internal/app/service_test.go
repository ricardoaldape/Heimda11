package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/core"
	"github.com/ricardoaldape/Heimda11/internal/secure"
	"github.com/ricardoaldape/Heimda11/internal/store"
)

func testService(t *testing.T) *Service {
	t.Helper()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil { t.Fatal(err) }
	cipher, err := secure.NewCipher(strings.Repeat("11",32))
	if err != nil { t.Fatal(err) }
	return NewService(st,cipher,"pro")
}

func createTestAgent(t *testing.T,s *Service,name string) core.AgentCreateResult {
	t.Helper()
	result,err:=s.CreateAgent(core.Agent{Name:name,Role:"worker",Department:"ops",HumanOwner:"human",AutonomyLevel:2,MonthlyBudget:0.10,AllowedTools:[]string{"stripe","crm","llm"}})
	if err!=nil{t.Fatal(err)}
	return result
}

func TestGateDefaultDenyAndApproval(t *testing.T){
	s:=testService(t)
	agent:=createTestAgent(t,s,"collector")
	result,err:=s.Evaluate(core.ActionRequest{AgentID:agent.Agent.ID,Tool:"stripe",Action:"refund",AmountUSD:100})
	if err!=nil{t.Fatal(err)}
	if result.Decision!=core.DecisionDeny{t.Fatalf("expected deny, got %s",result.Decision)}

	_,err=s.CreatePolicy(core.Policy{Name:"refund approval",Priority:100,AgentID:agent.Agent.ID,Tool:"stripe",Action:"refund",MinAmountUSD:500,Effect:core.DecisionApproval})
	if err!=nil{t.Fatal(err)}
	result,err=s.Evaluate(core.ActionRequest{AgentID:agent.Agent.ID,Tool:"stripe",Action:"refund",AmountUSD:800})
	if err!=nil{t.Fatal(err)}
	if result.Decision!=core.DecisionApproval||result.ApprovalID==""{t.Fatalf("unexpected result: %+v",result)}
	if _,err:=s.ResolveApproval(result.ApprovalID,core.ApprovalApproved,"finance-manager","approved");err!=nil{t.Fatal(err)}
}

func TestGateRejectsToolOutsideAllowList(t *testing.T){
	s:=testService(t)
	agent:=createTestAgent(t,s,"sales")
	_,err:=s.CreatePolicy(core.Policy{Name:"global allow",Priority:1,AgentID:"*",Tool:"*",Action:"*",Effect:core.DecisionAllow})
	if err!=nil{t.Fatal(err)}
	result,err:=s.Evaluate(core.ActionRequest{AgentID:agent.Agent.ID,Tool:"database",Action:"drop"})
	if err!=nil{t.Fatal(err)}
	if result.Decision!=core.DecisionDeny{t.Fatalf("expected deny, got %s",result.Decision)}
}

func TestMeterBudgetAndPerformance(t *testing.T){
	s:=testService(t)
	agent:=createTestAgent(t,s,"analyst")
	_,total,over,err:=s.RecordUsage(core.UsageRecord{AgentID:agent.Agent.ID,Provider:"local",Model:"test",InputTokens:100,OutputTokens:20,CostUSD:0.11})
	if err!=nil{t.Fatal(err)}
	if !over||total<0.109{t.Fatalf("expected over budget, total=%v over=%v",total,over)}
	if _,err:=s.RecordEvent(core.Event{AgentID:agent.Agent.ID,Type:"task.completed",Status:"success",ValueUSD:2});err!=nil{t.Fatal(err)}
	p,err:=s.Performance(agent.Agent.ID)
	if err!=nil{t.Fatal(err)}
	if p.Tasks!=1||p.Successes!=1||p.CostUSD<0.109||p.ValueUSD!=2{t.Fatalf("unexpected performance: %+v",p)}
	if p.ROI<=0{t.Fatalf("expected positive ROI, got %v",p.ROI)}
}

func TestVaultLeaseOneTimeAndScoped(t *testing.T){
	s:=testService(t)
	a:=createTestAgent(t,s,"a")
	b:=createTestAgent(t,s,"b")
	secret,err:=s.CreateSecret("provider","secret-value",a.Agent.ID)
	if err!=nil{t.Fatal(err)}
	if _,err:=s.CreateLease(secret.ID,b.Agent.ID,time.Minute);err==nil{t.Fatal("expected scoped secret rejection")}
	lease,err:=s.CreateLease(secret.ID,a.Agent.ID,time.Minute)
	if err!=nil{t.Fatal(err)}
	value,err:=s.RedeemLease(lease.ID,a.Agent.ID)
	if err!=nil{t.Fatal(err)}
	if value!="secret-value"{t.Fatalf("got %q",value)}
	if _,err:=s.RedeemLease(lease.ID,a.Agent.ID);err==nil{t.Fatal("expected second redemption to fail")}
}

func TestMemoryACL(t *testing.T){
	s:=testService(t)
	a:=createTestAgent(t,s,"a")
	b:=createTestAgent(t,s,"b")
	_,err:=s.CreateMemoryItem(core.MemoryItem{Namespace:"company",Title:"Policy",Content:"secret policy",AllowedAgentIDs:[]string{a.Agent.ID}})
	if err!=nil{t.Fatal(err)}
	if got:=s.SearchMemory("policy","company",a.Agent.ID);len(got)!=1{t.Fatalf("authorized agent got %d",len(got))}
	if got:=s.SearchMemory("policy","company",b.Agent.ID);len(got)!=0{t.Fatalf("unauthorized agent got %d",len(got))}
}

func TestFlowRequiresGateForEveryStep(t *testing.T){
	s:=testService(t)
	a:=createTestAgent(t,s,"flow")
	_,err:=s.CreatePolicy(core.Policy{Name:"crm allow",Priority:10,AgentID:a.Agent.ID,Tool:"crm",Action:"update",Effect:core.DecisionAllow})
	if err!=nil{t.Fatal(err)}
	flow,err:=s.CreateFlow(core.FlowDefinition{Name:"test",Steps:[]core.FlowStep{{Name:"one",AgentID:a.Agent.ID,Tool:"crm",Action:"update"},{Name:"two",AgentID:a.Agent.ID,Tool:"crm",Action:"update"}}})
	if err!=nil{t.Fatal(err)}
	run,err:=s.StartFlow(flow.ID)
	if err!=nil{t.Fatal(err)}
	if run.Status!="needs_gate"{t.Fatalf("expected needs_gate, got %s",run.Status)}
	if _,err:=s.CompleteFlowStep(run.ID,"success","bypass");err==nil{t.Fatal("expected pre-gate completion to fail")}

	run,step,gate,err:=s.CurrentFlowStep(run.ID)
	if err!=nil{t.Fatal(err)}
	if step==nil||gate==nil||gate.Decision!=core.DecisionAllow||run.Status!="running"{t.Fatalf("unexpected current: %+v %+v %+v",run,step,gate)}
	run,err=s.CompleteFlowStep(run.ID,"success","done")
	if err!=nil{t.Fatal(err)}
	if run.Status!="needs_gate"{t.Fatalf("next step should need gate, got %s",run.Status)}
	if _,err:=s.CompleteFlowStep(run.ID,"success","bypass second");err==nil{t.Fatal("expected second pre-gate completion to fail")}
}

func TestRelayFallsBackAndUsesVaultSecret(t *testing.T){
	s:=testService(t)
	a:=createTestAgent(t,s,"relay")
	secret,err:=s.CreateSecret("provider-key","abc123","")
	if err!=nil{t.Fatal(err)}
	_,err=s.CreatePolicy(core.Policy{Name:"relay allow",Priority:100,AgentID:a.Agent.ID,Tool:"llm",Action:"chat.completions",Effect:core.DecisionAllow})
	if err!=nil{t.Fatal(err)}

	good:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.URL.Path!="/chat/completions"{http.NotFound(w,r);return}
		if r.Header.Get("Authorization")!="Bearer abc123"{http.Error(w,"bad auth",http.StatusUnauthorized);return}
		w.Header().Set("Content-Type","application/json")
		_,_=w.Write([]byte(`{"ok":true}`))
	}))
	defer good.Close()

	_,err=s.CreateProvider(core.RelayProvider{Name:"broken",BaseURL:"http://127.0.0.1:1",SecretID:secret.ID,Priority:1,TimeoutSeconds:1})
	if err!=nil{t.Fatal(err)}
	_,err=s.CreateProvider(core.RelayProvider{Name:"good",BaseURL:good.URL,SecretID:secret.ID,Priority:2,TimeoutSeconds:2})
	if err!=nil{t.Fatal(err)}

	status,body,provider,err:=s.RelayChat(context.Background(),core.RelayRequest{AgentID:a.Agent.ID,Payload:[]byte(`{"messages":[]}`)})
	if err!=nil{t.Fatal(err)}
	if status!=200||provider!="good"||!strings.Contains(string(body),`"ok":true`){t.Fatalf("status=%d provider=%s body=%s",status,provider,body)}
}


func TestCredentialRotationRevokesOldKey(t *testing.T){
	s:=testService(t)
	a:=createTestAgent(t,s,"rotate")
	if id,ok:=s.AuthenticateAgentKey(a.APIKey);!ok||id!=a.Agent.ID{t.Fatal("initial credential should authenticate")}
	next,err:=s.RotateCredential(a.Agent.ID)
	if err!=nil{t.Fatal(err)}
	if _,ok:=s.AuthenticateAgentKey(a.APIKey);ok{t.Fatal("old credential remained valid after rotation")}
	if id,ok:=s.AuthenticateAgentKey(next);!ok||id!=a.Agent.ID{t.Fatal("new credential should authenticate")}
	if _,err:=s.SetAgentStatus(a.Agent.ID,core.AgentSuspended);err!=nil{t.Fatal(err)}
	if _,ok:=s.AuthenticateAgentKey(next);ok{t.Fatal("suspended agent credential should not authenticate")}
}
