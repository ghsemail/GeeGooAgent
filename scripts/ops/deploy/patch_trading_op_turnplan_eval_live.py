#!/usr/bin/env python3
"""Patch trading_operation main.dart.js for per-case TurnPlan live eval (Chat + verify)."""
from __future__ import annotations

import argparse
import os
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path

BRANCH_OLD = 's=a==="hello_then_stock_analysis"?3:a==="turn_plan_routing"?9:4'
BRANCH_NEW = 's=a==="hello_then_stock_analysis"?3:B.c.aS(a,"turn_plan_")?9:4'

ATQ2_CLASS = r"""aTQ2:function aTQ2(a,b,c,d,e,f){this.a=a
this.b=b
this.c=c
this.d=d
this.e=e
this.f=f},"""

ATQ2_CTOR_ANCHOR = "arn:function arn(a,b){this.a=a"

ATQ2_PROTO = r"""A.aTQ2.prototype={
$1(a){return this.anq(a)},
anq(a){var s=0,r=A.u(t.y),q,p=this,o,n,m,l,k,j,i,h,g,f
var $async$$1=A.p(function(b,c){if(b===1)return A.q(c,r)
for(;;)switch(s){case 0:l=p.a
k=a.a
j=a.b
h=p.d
o=J.aT(h)
case 2:if(o.v()){f=o.gO(o)
n=J.v(f)
l.fd(B.co,j+" setup: "+A.k(n),p.c)
s=3
return A.i(l.aSj(k,20,j+" setup",p.c,n),$async$$1)}case 3:if(!c){q=!1
s=1
break}s=2
break
m=J.v(p.e)
l.fd(B.co,j+" send: "+A.k(m),p.c)
s=4
return A.i(l.aSj(k,20,j+" turn",p.c,m),$async$$1)
case 4:if(!c){q=!1
s=1
break}i=k.cy.gh(0)
if(i==null||J.aL(i)===0)throw A.o(A.aF("missing session_id"))
s=5
return A.i(p.f.a.hh(A.df()+"/v1/dashboard/eval/cases/"+p.c+"/verify",A.m(["session_id",i],t.N,t.z),A.cF(p.f.dh(),null,null),t.z),$async$$1)
case 5:o=A.ao(t.f.a(c.a),t.N,t.z)
q=J.f(J.F(o,"ok"),!1)
if(!q){n=J.F(o,"detail")
throw A.o(A.aF(n==null?"verify failed":J.v(n)))}s=1
break
case 1:return A.r(q,r)}})
return A.t($async$$1,r)},
$S:277}"""

UF_METHOD = r"""uF(a){return this.aTQ(a)},
aTQ(b4){var s=0,r=A.u(t.H),q=1,p=[],o=[],n=this,m,l,k,j,i,h,g,f,e,d,c,b,a,a0,a1,a2,a3,a4,a5,a6,a7,a8,a9,b0,b1,b2,b3,b4c,b4d
var $async$uF=A.p(function(b5,b6){if(b5===1){p.push(b6)
s=q}for(;;)switch(s){case 0:a7=n.kd(b4)
a8=new A.c1(Date.now(),0,!1)
b0="run-"+a8.a
$.aj()
a=t.C
a0=$.c
if(a0==null)a0=$.c=B.d
if($.bG.a5(0,a0.bq(0,A.aW(a),null))){a0=$.c;(a0==null?$.c=B.d:a0).k(0,null,a).nI(B.et)}s=2
return A.i(A.ph(),$async$uF)
case 2:m=n.a6Q()
a=B.c.a0(J.v(a8),0,19)
a1=A.bA_(b4,!1,null,null,null,b0,null,m.p2.gh(0),null,a8,B.l9,a7.b+" \xb7 "+a)
n.go=a1
a2=n.ay
a2.hw(a2,0,a1)
n.ch.sh(0,b0)
a1=n.go
a1.toString
s=3
return A.i(n.rn(a1),$async$uF)
case 3:a1=n.CW
a1.sh(0,!0)
a0=n.cx
a3=a0.Y$
a3===$&&A.d()
J.dc(a3,b4,new A.pi(B.l9,null,null))
a3=a0.a1$
a4=a0.gh(0)
a3.r=a4
a3.U(a4)
a4=a0.gh(0)
a3.r=a4
a3.U(a4)
n.fd(B.co,"======== \u5f00\u59cb TurnPlan \u8bc4\u4f30\uff08\u771f\u5b9e Chat\uff09========",b4)
q=5
h=$.bEP()
g=t.f
f=t.z
e=A
s=7
return A.i(h.eB(),$async$uF)
case 7:s=8
return A.i(h.a.hE(0,A.df()+"/v1/dashboard/eval/cases/"+b4,A.cF(h.dh(),null,null),f),$async$uF)
case 8:b4c=e.ao(g.a(b6.a),t.N,f)
b4d=J.F(b4c,"case")
if(b4d==null)throw A.o(A.aF("eval case not found"))
b2=J.F(b4d,"utterance")
if(b2==null||J.aL(J.v(b2))===0){b3=J.F(b4d,"options")
if(t.f.b(b3))b2=J.F(b3,"message")}
if(b2==null||J.aL(J.v(b2))===0)throw A.o(A.aF("missing utterance"))
b3=J.F(b4d,"setup_utterances")
if(b3==null&&t.f.b(J.F(b4d,"options")))b3=J.F(J.F(b4d,"options"),"setup_messages")
a3=t.Ch
j=A.b([new A.ua(m,"Dock")],a3)
for(a3=j,a4=a3.length,a5=0;a5<a3.length;a3.length===a4||(0,A.Q)(a3),++a5){i=a3[a5]
if(a7.e===B.hY){i.a.v1()
n.fd(B.co,i.b+"\uff1a\u5df2\u6e05\u7a7a Chat \u4f1a\u8bdd",b4)}}a3=j
s=9
return A.i(A.ku(new A.E(a3,new A.aTQ2(n,a7,b4,b3,b2,h),A.U(a3).j("E<1,aU<K>>")),t.y),$async$uF)
case 9:b1=b6
if(J.bzR(b1,new A.arq())){a3=A.aF("TurnPlan verify failed")
throw A.o(a3)}g=B.p.dK(new A.c1(Date.now(),0,!1).fe(a8).a,1000)
a3=n.go
a3.f=B.la
a3.e=new A.c1(Date.now(),0,!1)
a3.y=g
J.dc(a0.Y$,b4,new A.pi(B.la,g,null))
a3=a0.gh(0)
a3.r=a3
a3.U(a3)
n.fd(B.nE,"TurnPlan \u8bc4\u4f30\u901a\u8fc7\uff08"+B.h.W(g/1000,1)+"s\uff09",b4)
o.push(11)
s=10
break
case 5:q=4
b1=p.pop()
f=A.am(b1)
e=B.p.dK(new A.c1(Date.now(),0,!1).fe(a8).a,1000)
a3=n.go
a3.f=B.lb
a3.e=new A.c1(Date.now(),0,!1)
a3.y=e
a3.z=J.v(f)
J.dc(a0.Y$,b4,new A.pi(B.lb,e,J.v(f)))
a3=a0.gh(0)
a3.r=a3
a3.U(a3)
n.fd(B.jb,"TurnPlan \u8bc4\u4f30\u5931\u8d25: "+A.k(f),b4)
o.push(11)
s=10
break
case 4:o=[1]
case 10:q=1
a1.sh(0,!1)
a3=a0.gh(0)
a3.r=a3
a3.U(a3)
if(a7.e===B.nF){m.v1()
n.fd(B.co,"Chat \u4f1a\u8bdd\u5df2\u5728\u8fd0\u884c\u540e\u6e05\u7a7a",b4)}n.fd(B.co,"======== TurnPlan \u8bc4\u4f30\u7ed3\u675f ========",b4)
b=n.go
n.go=null
s=b!=null?12:13
break
case 12:s=14
return A.i(n.rn(b),$async$uF)
case 14:case 13:a=a2.a1$
a2=a2.gh(0)
a.r=a2
a.U(a2)
s=o.pop()
break
case 11:return A.r(null,r)
case 1:return A.q(p.at(-1),r)}})
return A.t($async$uF,r)},"""

UF_ANCHOR_START = "uF(a){return this.aTQ(a)},"
UF_ANCHOR_END = "return A.t($async$uF,r)},\nAY(a){return this.apA(a)},"
UF_TAIL = "AY(a){return this.apA(a)},"
DUPLICATE_UF_END = "return A.t($async$uF,r)},\nreturn A.t($async$uF,r)},\n"

ATQ2_INSERT_BEFORE = "A.arn.prototype={"


def repair_broken_uf_patch(content: str) -> str:
    if DUPLICATE_UF_END in content:
        return content.replace(DUPLICATE_UF_END, "return A.t($async$uF,r)},\n", 1)
    return content


def is_fully_patched(content: str) -> bool:
    if DUPLICATE_UF_END in content:
        return False
    if "$S:197}\naTQ2:function aTQ2" in content:
        return False
    return (
        "A.aTQ2.prototype" in content
        and 'B.c.aS(a,"turn_plan_")?9' in content
        and "/verify" in content
        and ATQ2_CTOR_ANCHOR in content
    )


def patch_main_dart_js(content: str) -> str:
    content = repair_broken_uf_patch(content)
    if is_fully_patched(content):
        return content

    if BRANCH_OLD in content:
        content = content.replace(BRANCH_OLD, BRANCH_NEW, 1)
    elif BRANCH_NEW not in content:
        raise RuntimeError("bb3 branch pattern not found in main.dart.js")

    if ATQ2_CTOR_ANCHOR not in content:
        raise RuntimeError("aTQ2 ctor anchor not found")
    if "aTQ2:function aTQ2(a,b,c,d,e,f)" not in content:
        content = content.replace(
            ATQ2_CTOR_ANCHOR,
            ATQ2_CLASS + "\n" + ATQ2_CTOR_ANCHOR,
            1,
        )

    if ATQ2_INSERT_BEFORE not in content:
        raise RuntimeError("aTQ2 insert anchor not found")
    if "A.aTQ2.prototype" not in content:
        content = content.replace(
            ATQ2_INSERT_BEFORE,
            ATQ2_PROTO + "\n" + ATQ2_INSERT_BEFORE,
            1,
        )

    start = content.find(UF_ANCHOR_START)
    if start < 0:
        raise RuntimeError("uF anchor not found")
    end = content.find(UF_ANCHOR_END, start)
    if end < 0:
        raise RuntimeError("uF end anchor not found")
    content = content[:start] + UF_METHOD + "\n" + UF_TAIL + "\n" + content[end + len(UF_ANCHOR_END) :]
    return content


def patch_file(path: Path) -> bool:
    original = path.read_text(encoding="utf-8")
    patched = patch_main_dart_js(original)
    if patched == original:
        print(f"already patched: {path}")
        return False
    backup = path.with_suffix(path.suffix + ".bak-" + datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S"))
    shutil.copy2(path, backup)
    path.write_text(patched, encoding="utf-8")
    print(f"patched: {path}")
    print(f"backup: {backup}")
    return True


def ssh_run(client, cmd: str, timeout: int = 120) -> str:
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    if out.strip():
        print(out.rstrip())
    if err.strip():
        print("STDERR:", err.rstrip())
    return out


def deploy_remote(host: str, user: str, password: str, web_dir: str) -> int:
    import paramiko

    remote_js = f"{web_dir}/main.dart.js"
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, username=user, password=password, timeout=20)

    script_path = Path(__file__).resolve()
    sftp = client.open_sftp()
    remote_script = "/tmp/patch_trading_op_turnplan_eval_live.py"
    sftp.put(str(script_path), remote_script)
    sftp.close()

    ts = datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")
    restore_backup = f"{remote_js}.bak-20260905013404"
    ssh_run(client, f"cp {remote_js} {remote_js}.bak-{ts}")
    ssh_run(
        client,
        f"test -f {restore_backup} && cp {restore_backup} {remote_js} || true",
    )
    ssh_run(client, f"python3 {remote_script} --file {remote_js}")

    print("=== sync web -> nginx container ===")
    ssh_run(
        client,
        "NGINX=$(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'); "
        f"echo nginx=$NGINX; "
        f"docker cp {web_dir}/. ${{NGINX}}:/usr/share/nginx/html/; "
        'docker exec $NGINX sh -c \'grep -c "turn_plan_" /usr/share/nginx/html/main.dart.js; '
        'grep -c aTQ2 /usr/share/nginx/html/main.dart.js\'',
    )
    client.close()
    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--file", type=Path, help="local main.dart.js to patch in place")
    parser.add_argument("--deploy", action="store_true", help="patch and deploy to trading-operation host")
    parser.add_argument("--host", default=os.environ.get("TRADING_OP_SSH_HOST", "146.56.225.252"))
    parser.add_argument("--user", default=os.environ.get("TRADING_OP_SSH_USER", "root"))
    parser.add_argument(
        "--password",
        default=os.environ.get("TRADING_OP_SSH_PASSWORD") or os.environ.get("GEEGOO_AGENT_SSH_PASSWORD"),
    )
    parser.add_argument("--web-dir", default="/root/apps/trading_operation/web")
    args = parser.parse_args()

    if args.file:
        patch_file(args.file)
        return 0

    if args.deploy:
        if not args.password:
            print("missing SSH password", file=sys.stderr)
            return 1
        return deploy_remote(args.host, args.user, args.password, args.web_dir)

    parser.print_help()
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
