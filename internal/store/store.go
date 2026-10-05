package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/ricardoaldape/Heimda11/internal/core"
)

var ErrNotFound = errors.New("not found")

type State struct {
	Agents         map[string]core.Agent          `json:"agents"`
	Credentials    map[string]string              `json:"credential_hashes"`
	Policies       map[string]core.Policy         `json:"policies"`
	Approvals      map[string]core.Approval       `json:"approvals"`
	Usage          []core.UsageRecord             `json:"usage"`
	Events         []core.Event                   `json:"events"`
	Secrets        map[string]core.SecretRecord   `json:"secrets"`
	Leases         map[string]core.SecretLease    `json:"leases"`
	Providers      map[string]core.RelayProvider  `json:"providers"`
	Memory         map[string]core.MemoryItem     `json:"memory"`
	Certifications map[string]core.Certification  `json:"certifications"`
	Flows          map[string]core.FlowDefinition `json:"flows"`
	FlowRuns       map[string]core.FlowRun        `json:"flow_runs"`
}

type FileStore struct {
	mu    sync.RWMutex
	path  string
	state State
}

func NewFileStore(path string) (*FileStore, error) {
	s := &FileStore{path: path}
	s.normalize()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &s.state); err != nil {
			return nil, err
		}
	}
	s.normalize()
	return s, nil
}

func (s *FileStore) normalize() {
	if s.state.Agents == nil { s.state.Agents = map[string]core.Agent{} }
	if s.state.Credentials == nil { s.state.Credentials = map[string]string{} }
	if s.state.Policies == nil { s.state.Policies = map[string]core.Policy{} }
	if s.state.Approvals == nil { s.state.Approvals = map[string]core.Approval{} }
	if s.state.Usage == nil { s.state.Usage = []core.UsageRecord{} }
	if s.state.Events == nil { s.state.Events = []core.Event{} }
	if s.state.Secrets == nil { s.state.Secrets = map[string]core.SecretRecord{} }
	if s.state.Leases == nil { s.state.Leases = map[string]core.SecretLease{} }
	if s.state.Providers == nil { s.state.Providers = map[string]core.RelayProvider{} }
	if s.state.Memory == nil { s.state.Memory = map[string]core.MemoryItem{} }
	if s.state.Certifications == nil { s.state.Certifications = map[string]core.Certification{} }
	if s.state.Flows == nil { s.state.Flows = map[string]core.FlowDefinition{} }
	if s.state.FlowRuns == nil { s.state.FlowRuns = map[string]core.FlowRun{} }
}

func (s *FileStore) persistLocked() error {
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil { return err }
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil { return err }
	return os.Rename(tmp, s.path)
}

func (s *FileStore) UpsertAgent(v core.Agent) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Agents[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Agent(id string) (core.Agent,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Agents[id]; if !ok{return core.Agent{},ErrNotFound}; return v,nil }
func (s *FileStore) Agents() []core.Agent { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.Agent,0,len(s.state.Agents)); for _,v:=range s.state.Agents{out=append(out,v)}; sort.Slice(out,func(i,j int)bool{return out[i].CreatedAt.Before(out[j].CreatedAt)}); return out }
func (s *FileStore) ReplaceCredential(hash, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for existingHash, existingAgentID := range s.state.Credentials {
		if existingAgentID == agentID {
			delete(s.state.Credentials, existingHash)
		}
	}
	s.state.Credentials[hash] = agentID
	return s.persistLocked()
}
func (s *FileStore) CredentialAgent(hash string) (string,bool) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Credentials[hash]; return v,ok }

func (s *FileStore) UpsertPolicy(v core.Policy) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Policies[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Policies() []core.Policy { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.Policy,0,len(s.state.Policies)); for _,v:=range s.state.Policies{out=append(out,v)}; sort.SliceStable(out,func(i,j int)bool{if out[i].Priority==out[j].Priority{return out[i].CreatedAt.Before(out[j].CreatedAt)};return out[i].Priority>out[j].Priority}); return out }

func (s *FileStore) PutApproval(v core.Approval) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Approvals[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Approval(id string) (core.Approval,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Approvals[id]; if !ok{return core.Approval{},ErrNotFound}; return v,nil }
func (s *FileStore) Approvals() []core.Approval { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.Approval,0,len(s.state.Approvals)); for _,v:=range s.state.Approvals{out=append(out,v)}; sort.Slice(out,func(i,j int)bool{return out[i].RequestedAt.After(out[j].RequestedAt)}); return out }

func (s *FileStore) AddUsage(v core.UsageRecord) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Usage=append(s.state.Usage,v); return s.persistLocked() }
func (s *FileStore) Usage(agentID string) []core.UsageRecord { s.mu.RLock(); defer s.mu.RUnlock(); out:=[]core.UsageRecord{}; for _,v:=range s.state.Usage{if agentID==""||v.AgentID==agentID{out=append(out,v)}}; return out }

func (s *FileStore) AddEvent(v core.Event) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Events=append(s.state.Events,v); return s.persistLocked() }
func (s *FileStore) Events(agentID string) []core.Event { s.mu.RLock(); defer s.mu.RUnlock(); out:=[]core.Event{}; for _,v:=range s.state.Events{if agentID==""||v.AgentID==agentID{out=append(out,v)}}; return out }

func (s *FileStore) PutSecret(v core.SecretRecord) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Secrets[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Secret(id string) (core.SecretRecord,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Secrets[id]; if !ok{return core.SecretRecord{},ErrNotFound}; return v,nil }
func (s *FileStore) Secrets() []core.SecretRecord { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.SecretRecord,0,len(s.state.Secrets)); for _,v:=range s.state.Secrets{out=append(out,v)}; return out }
func (s *FileStore) PutLease(v core.SecretLease) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Leases[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Lease(id string) (core.SecretLease,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Leases[id]; if !ok{return core.SecretLease{},ErrNotFound}; return v,nil }

func (s *FileStore) PutProvider(v core.RelayProvider) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Providers[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Provider(id string) (core.RelayProvider,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Providers[id]; if !ok{return core.RelayProvider{},ErrNotFound}; return v,nil }
func (s *FileStore) Providers() []core.RelayProvider { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.RelayProvider,0,len(s.state.Providers)); for _,v:=range s.state.Providers{out=append(out,v)}; sort.SliceStable(out,func(i,j int)bool{return out[i].Priority<out[j].Priority}); return out }

func (s *FileStore) PutMemory(v core.MemoryItem) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Memory[v.ID]=v; return s.persistLocked() }
func (s *FileStore) MemoryItems() []core.MemoryItem { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.MemoryItem,0,len(s.state.Memory)); for _,v:=range s.state.Memory{out=append(out,v)}; return out }

func (s *FileStore) PutCertification(v core.Certification) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Certifications[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Certifications(agentID string) []core.Certification { s.mu.RLock(); defer s.mu.RUnlock(); out:=[]core.Certification{}; for _,v:=range s.state.Certifications{if agentID==""||v.AgentID==agentID{out=append(out,v)}}; return out }

func (s *FileStore) PutFlow(v core.FlowDefinition) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.Flows[v.ID]=v; return s.persistLocked() }
func (s *FileStore) Flow(id string) (core.FlowDefinition,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.Flows[id]; if !ok{return core.FlowDefinition{},ErrNotFound}; return v,nil }
func (s *FileStore) Flows() []core.FlowDefinition { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.FlowDefinition,0,len(s.state.Flows)); for _,v:=range s.state.Flows{out=append(out,v)}; return out }
func (s *FileStore) PutFlowRun(v core.FlowRun) error { s.mu.Lock(); defer s.mu.Unlock(); s.state.FlowRuns[v.ID]=v; return s.persistLocked() }
func (s *FileStore) FlowRun(id string) (core.FlowRun,error) { s.mu.RLock(); defer s.mu.RUnlock(); v,ok:=s.state.FlowRuns[id]; if !ok{return core.FlowRun{},ErrNotFound}; return v,nil }
func (s *FileStore) FlowRuns() []core.FlowRun { s.mu.RLock(); defer s.mu.RUnlock(); out:=make([]core.FlowRun,0,len(s.state.FlowRuns)); for _,v:=range s.state.FlowRuns{out=append(out,v)}; return out }

func (s *FileStore) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw,_:=json.Marshal(s.state)
	var out State
	_ = json.Unmarshal(raw,&out)
	return out
}
