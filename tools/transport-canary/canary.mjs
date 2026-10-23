// One disposable free-trial account and peer; no invoices, payments or OS routes.
// Configuration and private keys exist only in process memory and probe stdin.
import {inflateSync} from 'node:zlib';
import {createRequire} from 'node:module';
import {spawn} from 'node:child_process';
import {writeFileSync, existsSync} from 'node:fs';
import {fileURLToPath} from 'node:url';

const require = createRequire(new URL('./package.json', import.meta.url));
const {Wallet} = require('ethers');
const binary = fileURLToPath(new URL('./probe', import.meta.url));
const output = new URL('./result.json', import.meta.url);
const base = 'https://api.vpn.defiguru.ai', origin = 'https://defiguru.ai';
const result = {startedAt:new Date().toISOString(),testLabel:'Maintainer-triggered isolated transport canary',steps:[],ok:false,cleanupConfirmed:false,moneyMoved:false,invoicesCreated:false,globalNetworkChanged:false,configPersisted:false,locationClaim:'GitHub hosted Ubuntu network; independent of owner Mac route. No claim of direct Russia/Italy ISP or native app import.'};
const jar = new Map(), owned = new Set(), mutatingRequests = new Set();
let authenticated = false, disposableAccountConfirmed = false, accountId, child, cleanupFlight, stopping=false, cleaning=false;
function fail(code) {const error=new Error('sanitized-test-failure');error.safeCode=code;throw error}
function check(value,code) {if(!value)fail(code)}
function step(name) {result.steps.push(name);console.log(`PASS ${name}`)}
function save() {writeFileSync(output,JSON.stringify(result,null,2)+'\n',{mode:0o600})}
async function performApi(method,path,body,expected=200) {
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),30000);
 try {
  const response=await fetch(base+path,{method,redirect:'error',cache:'no-store',signal:controller.signal,headers:{Origin:origin,Accept:'application/json','Content-Type':'application/json','X-Requested-With':'XMLHttpRequest',Cookie:[...jar].map(([k,v])=>`${k}=${v}`).join('; '),...(accountId&&(path==='/auth/logout'||path==='/me'||path.startsWith('/me/'))?{'X-VPN-Expected-Account-ID':accountId}:{})},...(body===undefined?{}:{body:JSON.stringify(body)})});
  for(const cookie of response.headers.getSetCookie()){const pair=cookie.split(';')[0],at=pair.indexOf('=');if(at>0){const k=pair.slice(0,at),v=pair.slice(at+1);v?jar.set(k,v):jar.delete(k);if(k==='vpn_session'&&v)authenticated=true}}
  check(response.status===expected,`api-${method.toLowerCase()}-${path.split('/').filter(Boolean).slice(0,2).join('-')}-${response.status}`);
  check(response.headers.get('cache-control')?.includes('no-store'),'api-cache-policy');
  const raw=await response.text();check(Buffer.byteLength(raw)<1048576,'api-response-too-large');return JSON.parse(raw);
 } finally {clearTimeout(timer)}
}
async function api(method,path,body,expected=200,forCleanup=false) {
 if((stopping||cleaning)&&!forCleanup)fail('interrupted');
 const operation=performApi(method,path,body,expected);
 if(method!=='GET')mutatingRequests.add(operation);
 try{return await operation}finally{mutatingRequests.delete(operation)}
}
const cleanApi=(method,path,body,expected=200)=>api(method,path,body,expected,true);
async function probe(config) {
 return new Promise((resolve,reject)=>{
  child=spawn(binary,['--expected-egress=209.38.108.217','--duration-seconds=170'],{stdio:['pipe','pipe','pipe']});
  let stdout='',tooLarge=false;const timer=setTimeout(()=>{child.kill('SIGTERM');reject(Object.assign(new Error('sanitized'),{safeCode:'probe-timeout'}))},240000);
  child.stdout.on('data',data=>{if(stdout.length+data.length>131072){tooLarge=true;child.kill('SIGTERM')}else stdout+=data.toString()});
  child.stderr.on('data',()=>{});
  child.on('error',()=>{clearTimeout(timer);reject(Object.assign(new Error('sanitized'),{safeCode:'probe-process-unavailable'}))});
  child.on('close',code=>{
   clearTimeout(timer);child=null;
   try {
    check(!tooLarge,'probe-output-too-large');const raw=JSON.parse(stdout);
    const safe={};
    for(const key of ['ok','stage','code','handshake','txBytes','rxBytes','dnsViaTunnel','httpsViaTunnel','egressIPv4','egressMatchesExpected','httpsChecks','awgFields','mtu','durationMs','globalNetworkChanged','successfulSpanMs','handshakeCount','rekeyObserved','rekeyAdvanceMs','postRekeySuccessfulSamples','samples','handshakes','dnsChecks','failedSamples','successfulSamples','requestedSoakSeconds','requestTimeoutSeconds','dialMode']) if(raw[key]!==undefined)safe[key]=raw[key];
    result.transport=safe;check(code===0&&safe.ok===true,'transport-check-failed');check(safe.successfulSpanMs>=160000&&safe.rekeyObserved===true&&safe.handshakeCount>=2&&safe.postRekeySuccessfulSamples>=2,'sustained-rekey-required');
    check(safe.handshake===true&&safe.txBytes>0&&safe.rxBytes>0,'fresh-handshake-and-traffic-required');
    check(safe.dnsViaTunnel===true&&safe.httpsViaTunnel===true&&safe.httpsChecks>=2&&safe.egressIPv4==='209.38.108.217'&&safe.egressMatchesExpected===true,'tunnel-dns-and-egress-required');
    check(safe.globalNetworkChanged===false,'global-routing-changed');resolve(safe);
   } catch(error){reject(error)}
  });
  child.stdin.on('error',()=>{});child.stdin.end(config);
 });
}
async function verifyConnectionObservation(deviceId,probeStartedAt) {
 check(typeof accountId==='string'&&accountId.length>0,'connection-expected-account-required');
 const counter=value=>typeof value==='string'&&/^(0|[1-9][0-9]{0,19})$/.test(value)&&BigInt(value)<=(1n<<64n)-1n;
 // The global read cache may predate this new peer. Retry for at most 12.5s;
 // never create a replacement device or change the actual tunnel for this check.
 for(let attempt=0;attempt<6;attempt++) {
  const observation=await api('GET','/me/connection');
  check(observation&&Object.keys(observation).sort().join(',')==='devices,observedAt','connection-response-fields');
  check(Number.isSafeInteger(observation.observedAt)&&observation.observedAt>0&&observation.observedAt<=Date.now()+5000&&Date.now()-observation.observedAt<=45000,'connection-observation-time');
  check(Array.isArray(observation.devices)&&observation.devices.length===1&&observation.devices[0]?.deviceId===deviceId,'connection-owned-device-only');
  const device=observation.devices[0];
  check(Object.keys(device).sort().join(',')==='deviceId,downloadBytes,lastHandshakeAt,status,uploadBytes','connection-device-fields');
  check(['recent','idle','unknown'].includes(device.status),'connection-observation-status');
  if(device.status==='unknown')check(device.lastHandshakeAt===null&&device.downloadBytes===null&&device.uploadBytes===null,'connection-unknown-must-not-invent-metrics');
  else {
   check(counter(device.downloadBytes)&&counter(device.uploadBytes),'connection-exact-byte-counters');
   check(device.lastHandshakeAt===null||(Number.isSafeInteger(device.lastHandshakeAt)&&device.lastHandshakeAt>0&&device.lastHandshakeAt<=observation.observedAt),'connection-handshake-time');
  }
  if(device.status==='recent') {
   check(device.lastHandshakeAt!==null&&observation.observedAt-device.lastHandshakeAt<=180000&&device.lastHandshakeAt>=probeStartedAt-5000,'connection-fresh-probe-handshake');
   check(BigInt(device.downloadBytes)>0n&&BigInt(device.uploadBytes)>0n,'connection-real-transfer-counters');
   // Only the disposable device ID and public observation metrics are recorded.
   result.connectionObservation={observedAt:observation.observedAt,...device};
   result.connectionReadExpectedAccountBound=true;
   return;
  }
  if(attempt<5)await new Promise(resolve=>setTimeout(resolve,2500));
 }
 fail('connection-telemetry-not-recent');
}
async function cleanup() {
 if(cleanupFlight)return cleanupFlight;
 cleanupFlight=(async()=>{
  cleaning=true;
  await Promise.allSettled([...mutatingRequests]);
  let ok=true;
  if(authenticated){
   if(disposableAccountConfirmed&&accountId){
   try {const state=await cleanApi('GET','/me/status');for(const device of state.devices||[])owned.add(device.id)}catch{ok=false}
   for(const id of owned){
    let done=false;
    for(let attempt=0;attempt<6&&!done;attempt++){
     try {const removal=await cleanApi('DELETE',`/me/devices/${encodeURIComponent(id)}`);done=removal.ok===true&&removal.removal==='done'}catch{}
     if(!done)await new Promise(resolve=>setTimeout(resolve,1500));
    }
    if(!done)ok=false;
   }
   try {const after=await cleanApi('GET','/me/status');result.testDeviceSlotsAfterCleanup=after.devices.length;check(result.testDeviceSlotsAfterCleanup===0,'test-peer-removal-not-confirmed')}catch{ok=false}
   }
   try {await cleanApi('POST','/auth/logout',{});await cleanApi('GET','/me',undefined,401);authenticated=false;result.sessionRevoked=true}catch{ok=false}
  }
  jar.clear();result.cleanupConfirmed=ok;result.completedAt=new Date().toISOString();if(!ok)result.ok=false;save();
  console.log(ok?'PASS cleanup confirmed':'CLEANUP NOT CONFIRMED');return ok;
 })();return cleanupFlight;
}
for(const signal of ['SIGINT','SIGTERM'])process.once(signal,()=>{stopping=true;child?.kill('SIGTERM');void cleanup().finally(()=>process.exit(130))});

if(!process.argv.includes('--execute')){console.log('PLAN ONLY: free trial, authenticated ordinary 20% referral policy and inline readiness, one test peer, isolated user-space AWG DNS/HTTPS, protected own-device connection telemetry, peer removal and logout.');process.exit(0)}
try {
 check(existsSync(binary),'probe-binary-not-ready');
 const plans=await api('GET','/plans');check(plans.trial?.days===7&&plans.trial?.maxDevices===3,'trial-policy');step('public seven-day trial');
 const wallet=Wallet.createRandom();const challenge=await api('POST','/auth/wallet/challenge',{address:wallet.address,chainId:1});
 check(typeof challenge.message==='string'&&challenge.message.startsWith(`defiguru.ai wants you to sign in with your Ethereum account:\n${wallet.address}`)&&challenge.message.includes('URI: https://defiguru.ai/services/vpn\n'),'signed-login-origin');
 const signature=await wallet.signMessage(challenge.message);await api('POST','/auth/wallet/verify',{challengeId:challenge.challengeId,signature});authenticated=true;
 const me=await api('GET','/me');check(me.walletAddress?.toLowerCase()===wallet.address.toLowerCase(),'test-account-identity');accountId=me.accountId;result.ownedTestAccountId=accountId;
 const before=await api('GET','/me/status');check(before.status==='active'&&before.daysLeft===7&&before.maxDevices===3&&before.devices.length===0,'new-trial-entitlement');disposableAccountConfirmed=true;result.disposableAccountConfirmed=true;step('disposable signed account and free access');
 const referrals=await api('GET','/me/referrals');
 check(referrals.commissionPolicy?.enabled===true&&referrals.commissionPolicy?.rateBps===2000&&referrals.commissionPolicy?.renewals==='all'&&referrals.commissionPolicy?.basis==='gross-confirmed-amount','ordinary-referral-policy');
 check(referrals.inlineInvitesEnabled===true,'inline-referrals-ready');
 result.ordinaryReferralPolicy={enabled:true,rateBps:2000,renewals:'all',basis:'gross-confirmed-amount',inlineInvitesEnabled:true};
 step('authenticated ordinary referral policy stays 20% on all payments and renewals with inline invitations enabled');
 check(!stopping,'interrupted');let issued=await api('POST','/me/devices',{label:result.testLabel},201);
 check(issued.device?.id&&issued.device.status==='active'&&typeof issued.clientConfig==='string','acknowledged-peer');owned.add(issued.device.id);result.ownedTestDeviceIds=[...owned];save();
 check(!stopping,'interrupted');
 const config=issued.clientConfig;check(config.includes('[Interface]')&&config.includes('[Peer]')&&config.length<16384,'config-shape');
 check(typeof issued.amneziaKey==='string'&&/^vpn:\/\/[A-Za-z0-9_-]{8,32768}$/.test(issued.amneziaKey),'amnezia-key-required');
 const payload=Buffer.from(issued.amneziaKey.slice(6),'base64url');issued=null;
 check(payload.readUInt32BE(0)===Buffer.byteLength(config,'utf8')&&inflateSync(payload.subarray(4),{maxOutputLength:16384}).toString('utf8')===config,'amnezia-codec-roundtrip');step('acknowledged device and byte-exact Qt-compressed Amnezia key; not a native UI test');
 const probeStartedAt=Date.now();await probe(config);step('sustained real DNS and HTTPS through one tunnel, including verified rekey');
 await verifyConnectionObservation([...owned][0],probeStartedAt);step('protected own-device telemetry confirms a recent handshake and real transfer counters');
 const active=await api('GET','/me/status');check(active.devices.length===1&&active.devices[0].id===[...owned][0]&&active.devices[0].status==='active','test-device-status');result.ok=true;
} catch(error){result.failure=/^[a-z0-9-]{1,100}$/.test(error?.safeCode||'')?error.safeCode:'sanitized-probe-error';console.log(`FAIL ${result.failure}`);process.exitCode=1}
finally {if(!await cleanup())process.exitCode=1}
