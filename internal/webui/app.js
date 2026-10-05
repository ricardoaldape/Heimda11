const state={agents:[],token:sessionStorage.getItem("heimda11_admin_token")||""};
const $=id=>document.getElementById(id);
const money=n=>"$"+Number(n||0).toFixed(2);
const esc=v=>String(v??"").replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
const csv=v=>String(v||"").split(",").map(x=>x.trim()).filter(Boolean);

function notify(message,error=false){
  const el=$("toast");el.textContent=message;el.classList.remove("hidden","error");if(error)el.classList.add("error");
  clearTimeout(window.__toast);window.__toast=setTimeout(()=>el.classList.add("hidden"),4500);
}
async function api(path,options={}){
  const headers={"Content-Type":"application/json",...(options.headers||{})};
  if(state.token)headers.Authorization="Bearer "+state.token;
  const res=await fetch(path,{...options,headers});
  const text=await res.text();let data={};try{data=text?JSON.parse(text):{}}catch{data={raw:text}}
  if(!res.ok)throw new Error(data.error||res.statusText);
  return data;
}
function formJSON(form){
  const data=Object.fromEntries(new FormData(form).entries());return data;
}
function pill(value){return '<span class="pill '+esc(value)+'">'+esc(value)+'</span>'}

async function connect(){
  state.token=$("adminToken").value.trim();sessionStorage.setItem("heimda11_admin_token",state.token);
  await refreshAll();$("connectionState").textContent="Connected";notify("Connected to Heimda11");
}
async function refreshAll(){
  if(!state.token)return;
  const [meta,dashboard,agents,policies,approvals,secrets,providers,flows,runs]=await Promise.all([
    api("/v1/meta"),api("/v1/dashboard"),api("/v1/agents"),api("/v1/policies"),api("/v1/approvals"),
    api("/v1/vault/secrets"),api("/v1/relay/providers"),api("/v1/flows"),api("/v1/flows/runs")
  ]);
  state.agents=agents;
  $("editionBadge").textContent=meta.edition+" · v"+meta.version;
  $("mAgents").textContent=dashboard.agents_total;$("mActive").textContent=dashboard.agents_active;
  $("mApprovals").textContent=dashboard.pending_approvals;$("mBlocked").textContent=dashboard.blocked_total;
  $("mCost").textContent=money(dashboard.month_cost_usd);$("mValue").textContent=money(dashboard.month_value_usd);
  $("mSecrets").textContent=dashboard.secrets_total;$("mFlows").textContent=dashboard.flows_running;
  renderAgents(agents);renderPolicies(policies);renderApprovals(approvals);renderSecrets(secrets);renderProviders(providers);renderFlows(flows);renderRuns(runs);
  await renderPerformance(agents);
}
function renderAgents(items){
  $("agentsBody").innerHTML=items.map(a=>'<tr><td><div class="worker">'+esc(a.name)+'</div><div class="sub">'+esc(a.id)+'</div></td><td>'+esc(a.role)+'<div class="sub">'+esc(a.department||"—")+'</div></td><td>'+esc(a.human_owner)+'</td><td>L'+a.autonomy_level+'</td><td>'+money(a.monthly_budget_usd)+'</td><td>'+pill(a.status)+'</td><td><button data-status="'+esc(a.id)+'">'+(a.status==="active"?"Suspend":"Activate")+'</button></td></tr>').join("")||'<tr><td colspan="7" class="muted">No AI workers registered yet.</td></tr>';
  document.querySelectorAll("[data-status]").forEach(btn=>btn.onclick=async()=>{
    const a=state.agents.find(x=>x.id===btn.dataset.status);const status=a.status==="active"?"suspended":"active";
    await api("/v1/agents/"+a.id+"/status",{method:"POST",body:JSON.stringify({status})});notify("Agent status updated");await refreshAll();
  });
}
function renderPolicies(items){
  $("policiesList").innerHTML=items.map(p=>'<div class="item"><div class="top"><b>'+esc(p.name)+'</b>'+pill(p.effect)+'</div><div class="meta">'+esc(p.agent_id)+' · '+esc(p.tool)+"."+esc(p.action)+' · priority '+p.priority+(p.min_amount_usd?" · min "+money(p.min_amount_usd):"")+'</div></div>').join("")||'<div class="muted">No policies. Gate will default-deny.</div>';
}
function renderApprovals(items){
  $("approvalsList").innerHTML=items.map(a=>'<div class="item"><div class="top"><b>'+esc(a.tool)+"."+esc(a.action)+'</b>'+pill(a.status)+'</div><div class="meta">'+esc(a.agent_id)+(a.amount_usd?" · "+money(a.amount_usd):"")+' · '+new Date(a.requested_at).toLocaleString()+'</div>'+(a.status==="pending"?'<div class="item-actions"><button data-approve="'+esc(a.id)+'">Approve</button><button data-reject="'+esc(a.id)+'">Reject</button></div>':"")+'</div>').join("")||'<div class="muted">No approvals.</div>';
  document.querySelectorAll("[data-approve],[data-reject]").forEach(btn=>btn.onclick=async()=>{
    const id=btn.dataset.approve||btn.dataset.reject;const status=btn.dataset.approve?"approved":"rejected";
    await api("/v1/approvals/"+id+"/resolve",{method:"POST",body:JSON.stringify({status,resolved_by:"Heimda11 admin",note:"Resolved from control panel"})});
    notify("Approval "+status);await refreshAll();
  });
}
function renderSecrets(items){$("secretsList").innerHTML=items.map(s=>'<div class="item"><b>'+esc(s.name)+'</b><div class="meta">'+esc(s.id)+(s.scope_agent_id?" · "+esc(s.scope_agent_id):" · global")+'</div></div>').join("")||'<div class="muted">No secrets.</div>'}
function renderProviders(items){$("providersList").innerHTML=items.map(p=>'<div class="item"><div class="top"><b>'+esc(p.name)+'</b>'+pill(p.enabled?"active":"disabled")+'</div><div class="meta">'+esc(p.base_url)+' · priority '+p.priority+'</div></div>').join("")||'<div class="muted">No relay providers.</div>'}
function renderFlows(items){
  $("flowsList").innerHTML=items.map(f=>'<div class="item"><div class="top"><b>'+esc(f.name)+'</b><button data-startflow="'+esc(f.id)+'">Start</button></div><div class="meta">'+f.steps.length+' governed steps · '+esc(f.id)+'</div></div>').join("")||'<div class="muted">No flows.</div>';
  document.querySelectorAll("[data-startflow]").forEach(btn=>btn.onclick=async()=>{await api("/v1/flows/"+btn.dataset.startflow+"/runs",{method:"POST",body:"{}"});notify("Flow started");await refreshAll()});
}
function renderRuns(items){
  $("runsList").innerHTML=items.map(r=>'<div class="item"><div class="top"><b>'+esc(r.id)+'</b>'+pill(r.status)+'</div><div class="meta">flow '+esc(r.flow_id)+' · step '+r.current_step+' · '+r.history.length+' completed</div><div class="item-actions">'+(r.status==="running"||r.status==="waiting_approval"?'<button data-next="'+esc(r.id)+'">Inspect current step</button>':"")+'</div></div>').join("")||'<div class="muted">No workflow runs.</div>';
  document.querySelectorAll("[data-next]").forEach(btn=>btn.onclick=async()=>{const v=await api("/v1/flow-runs/"+btn.dataset.next+"/current");notify(v.gate?("Gate: "+v.gate.decision+" · "+v.gate.reason):"Flow has no current step");await refreshAll()});
}
async function renderPerformance(agents){
  const rows=[];
  for(const a of agents){
    try{const p=await api("/v1/agents/"+a.id+"/performance");rows.push('<tr><td>'+esc(a.name)+'</td><td>'+p.tasks+'</td><td>'+Math.round((p.success_rate||0)*100)+'%</td><td>'+money(p.cost_usd)+'</td><td>'+money(p.value_usd)+'</td><td>'+Number(p.roi||0).toFixed(2)+'x</td></tr>')}catch{}
  }
  $("performanceBody").innerHTML=rows.join("")||'<tr><td colspan="6" class="muted">No performance data yet.</td></tr>';
}

$("connectBtn").onclick=()=>connect().catch(e=>notify(e.message,true));
$("refreshBtn").onclick=()=>refreshAll().catch(e=>notify(e.message,true));
document.querySelectorAll(".nav").forEach(btn=>btn.onclick=()=>{
  document.querySelectorAll(".nav").forEach(x=>x.classList.remove("active"));btn.classList.add("active");
  document.querySelectorAll(".view").forEach(x=>x.classList.remove("active"));$(btn.dataset.view).classList.add("active");
  $("pageTitle").textContent=btn.textContent;
});

$("agentForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);d.autonomy_level=Number(d.autonomy_level);d.monthly_budget_usd=Number(d.monthly_budget_usd);d.allowed_tools=csv(d.allowed_tools);
  const r=await api("/v1/agents",{method:"POST",body:JSON.stringify(d)});
  $("newCredential").textContent="One-time agent API key: "+r.api_key;$("newCredential").classList.remove("hidden");e.target.reset();notify("AI worker registered");await refreshAll();
}catch(err){notify(err.message,true)}};
$("policyForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);d.priority=Number(d.priority);d.min_amount_usd=Number(d.min_amount_usd);d.max_amount_usd=Number(d.max_amount_usd);
  await api("/v1/policies",{method:"POST",body:JSON.stringify(d)});notify("Policy created");await refreshAll();
}catch(err){notify(err.message,true)}};
$("secretForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);await api("/v1/vault/secrets",{method:"POST",body:JSON.stringify(d)});e.target.reset();notify("Secret encrypted and stored");await refreshAll();
}catch(err){notify(err.message,true)}};
$("providerForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);d.priority=Number(d.priority);d.timeout_seconds=Number(d.timeout_seconds);
  await api("/v1/relay/providers",{method:"POST",body:JSON.stringify(d)});e.target.reset();notify("Relay provider added");await refreshAll();
}catch(err){notify(err.message,true)}};
$("memoryForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);d.tags=csv(d.tags);d.allowed_agent_ids=csv(d.allowed_agent_ids);
  await api("/v1/memory",{method:"POST",body:JSON.stringify(d)});e.target.reset();notify("Memory added");await searchMemory();
}catch(err){notify(err.message,true)}};
$("memorySearchBtn").onclick=()=>searchMemory().catch(e=>notify(e.message,true));
async function searchMemory(){
  const q=encodeURIComponent($("memoryQuery").value||"");const items=await api("/v1/memory?q="+q);
  $("memoryList").innerHTML=items.map(m=>'<div class="item"><b>'+esc(m.title)+'</b><div class="meta">'+esc(m.namespace)+' · '+esc((m.tags||[]).join(", "))+'</div><div class="meta">'+esc(m.content)+'</div></div>').join("")||'<div class="muted">No matches.</div>';
}
$("certForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);d.score=Number(d.score);
  await api("/v1/training/certifications",{method:"POST",body:JSON.stringify(d)});e.target.reset();notify("Certification issued");
}catch(err){notify(err.message,true)}};
$("flowForm").onsubmit=async e=>{e.preventDefault();try{
  const d=formJSON(e.target);let steps;try{steps=JSON.parse(d.steps)}catch{throw new Error("Steps must be valid JSON")}
  await api("/v1/flows",{method:"POST",body:JSON.stringify({name:d.name,steps})});e.target.reset();notify("Flow created");await refreshAll();
}catch(err){notify(err.message,true)}};

if(state.token){$("adminToken").value=state.token;refreshAll().then(()=>$("connectionState").textContent="Connected").catch(e=>{sessionStorage.removeItem("heimda11_admin_token");state.token="";notify(e.message,true)})}
