import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';

const source=readFileSync(new URL('./canary.mjs',import.meta.url),'utf8').replace(/^import .*;\n/gm,'');
for(const scenario of ['identity-mismatch','nonempty-baseline'])test(`cleanup never deletes a peer before disposable identity and empty baseline: ${scenario}`,async()=>{
 const calls=[],saved=[];let revoked=false;
 const wallet='0x'+'11'.repeat(20);
 const fetch=async(url,init)=>{
  const path=new URL(url).pathname,method=init.method;calls.push({method,path});let status=200,body={},cookies=[];
  if(path==='/plans')body={trial:{days:7,maxDevices:3}};
  else if(path==='/auth/wallet/challenge')body={challengeId:'synthetic',message:`defiguru.ai wants you to sign in with your Ethereum account:\n${wallet}\nURI: https://defiguru.ai/services/vpn\n`};
  else if(path==='/auth/wallet/verify')cookies=['vpn_session=synthetic; HttpOnly'];
  else if(path==='/me'){
   if(revoked){status=401;body={error:'unauthorized'}}
   else body={walletAddress:scenario==='identity-mismatch'?'0x'+'22'.repeat(20):wallet,accountId:'synthetic-existing-account'};
  }else if(path==='/me/status')body={status:'active',daysLeft:7,maxDevices:3,devices:[{id:'do-not-revoke-this-peer',status:'active'}]};
  else if(path==='/auth/logout'){revoked=true;cookies=['vpn_session=; Max-Age=0'];body={ok:true}}
  else if(method==='DELETE')throw Error('UNSAFE-PEER-DELETE');
  else throw Error('Unexpected route');
  return{status,headers:{getSetCookie:()=>cookies,get:()=> 'no-store'},text:async()=>JSON.stringify(body)};
 };
 const process={argv:['node','canary.mjs','--execute'],once(){},exit(){throw Error('unexpected exit')},exitCode:0};
 const context={Buffer,URL,Map,Set,AbortController,setTimeout,clearTimeout,Date,Promise,JSON,console:{log(){}},process,fetch,
  createRequire:()=>()=>({Wallet:{createRandom:()=>({address:wallet,signMessage:async()=> 'synthetic-signature'})}}),
  fileURLToPath:()=> '/synthetic-probe',existsSync:()=>true,writeFileSync:(_,value)=>saved.push(JSON.parse(value)),
  spawn:()=>{throw Error('probe must not start')},inflateSync:()=>{throw Error('codec must not start')}};
 const runnable=source.replaceAll('import.meta.url',JSON.stringify('file:///synthetic/canary.mjs'));
 await vm.runInNewContext('(async()=>{'+runnable+'\n})()',context);
 assert.equal(process.exitCode,1);
 assert.equal(calls.some(row=>row.method==='DELETE'),false);
 assert.equal(calls.filter(row=>row.path==='/me/status').length,scenario==='identity-mismatch'?0:1);
 assert.equal(revoked,true);
 assert.equal(saved.at(-1).cleanupConfirmed,true);
 assert.equal(saved.at(-1).ok,false);
});
