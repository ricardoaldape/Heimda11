package app

import (
	"errors"
	"strings"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/core"
)

func (s *Service) CreateSecret(name, value, scopeAgentID string) (core.SecretView, error) {
	name = strings.TrimSpace(name)
	if name == "" || value == "" {
		return core.SecretView{}, errors.New("name and value are required")
	}
	if scopeAgentID != "" {
		if _, err := s.store.Agent(scopeAgentID); err != nil { return core.SecretView{}, err }
	}
	encrypted, err := s.cipher.Encrypt(value)
	if err != nil { return core.SecretView{}, err }
	now := s.now()
	record := core.SecretRecord{
		ID: core.NewID("sec"), Name: name, ScopeAgentID: scopeAgentID,
		Ciphertext: encrypted, CreatedAt: now, RotatedAt: now,
	}
	if err := s.store.PutSecret(record); err != nil { return core.SecretView{}, err }
	_ = s.audit(scopeAgentID, "vault.secret_created", "success", "secret "+name+" created", map[string]any{"secret_id":record.ID})
	return secretView(record), nil
}

func (s *Service) RotateSecret(id, value string) (core.SecretView, error) {
	if value == "" { return core.SecretView{}, errors.New("value is required") }
	record, err := s.store.Secret(id)
	if err != nil { return core.SecretView{}, err }
	encrypted, err := s.cipher.Encrypt(value)
	if err != nil { return core.SecretView{}, err }
	record.Ciphertext = encrypted
	record.RotatedAt = s.now()
	if err := s.store.PutSecret(record); err != nil { return core.SecretView{}, err }
	_ = s.audit(record.ScopeAgentID, "vault.secret_rotated", "success", "secret rotated", map[string]any{"secret_id":record.ID})
	return secretView(record), nil
}

func secretView(v core.SecretRecord) core.SecretView {
	return core.SecretView{ID:v.ID,Name:v.Name,ScopeAgentID:v.ScopeAgentID,CreatedAt:v.CreatedAt,RotatedAt:v.RotatedAt}
}

func (s *Service) ListSecrets() []core.SecretView {
	records := s.store.Secrets()
	out := make([]core.SecretView,0,len(records))
	for _,record := range records { out=append(out,secretView(record)) }
	return out
}

func (s *Service) CreateLease(secretID, agentID string, ttl time.Duration) (core.SecretLease,error) {
	if ttl <= 0 || ttl > 15*time.Minute {
		return core.SecretLease{}, errors.New("lease TTL must be between 1 second and 15 minutes")
	}
	secret,err:=s.store.Secret(secretID)
	if err != nil { return core.SecretLease{},err }
	if _,err:=s.store.Agent(agentID); err != nil { return core.SecretLease{},err }
	if secret.ScopeAgentID!="" && secret.ScopeAgentID!=agentID {
		return core.SecretLease{}, errors.New("secret is scoped to a different agent")
	}
	now:=s.now()
	lease:=core.SecretLease{ID:core.NewID("lease"),SecretID:secretID,AgentID:agentID,ExpiresAt:now.Add(ttl),CreatedAt:now}
	if err:=s.store.PutLease(lease);err!=nil{return core.SecretLease{},err}
	_ = s.audit(agentID,"vault.lease_created","success","short-lived secret lease created",map[string]any{"secret_id":secretID,"lease_id":lease.ID})
	return lease,nil
}

func (s *Service) RedeemLease(leaseID, agentID string) (string,error) {
	lease,err:=s.store.Lease(leaseID)
	if err != nil{return "",err}
	if lease.AgentID!=agentID{return "",errors.New("lease belongs to another agent")}
	if lease.Redeemed{return "",errors.New("lease has already been redeemed")}
	if s.now().After(lease.ExpiresAt){return "",errors.New("lease has expired")}
	secret,err:=s.store.Secret(lease.SecretID)
	if err != nil{return "",err}
	value,err:=s.cipher.Decrypt(secret.Ciphertext)
	if err != nil{return "",err}
	now:=s.now()
	lease.Redeemed=true
	lease.RedeemedAt=&now
	if err:=s.store.PutLease(lease);err!=nil{return "",err}
	_ = s.audit(agentID,"vault.lease_redeemed","success","secret lease redeemed",map[string]any{"secret_id":secret.ID,"lease_id":lease.ID})
	return value,nil
}

func (s *Service) secretValueForAgent(secretID, agentID string) (string,error) {
	record,err:=s.store.Secret(secretID)
	if err != nil{return "",err}
	if record.ScopeAgentID!="" && record.ScopeAgentID!=agentID {
		return "", errors.New("secret is scoped to a different agent")
	}
	return s.cipher.Decrypt(record.Ciphertext)
}
