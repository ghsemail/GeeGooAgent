#!/usr/bin/env python3
"""Patch trading_operation main.dart.js to run TurnPlan eval via dashboard API."""
from __future__ import annotations

import argparse
import json
import os
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path

OLD_BB3_BRANCH = (
    'break}s=a==="hello_then_stock_analysis"?3:4\n'
    "break\n"
    "case 3:s=5\n"
    "return A.i(p.uC(),$async$G8)\n"
    "case 5:s=1\n"
    "break\n"
    "case 4:o=A.it(p.ax,new A.arw(a))\n"
    "if(o!=null){n=o.f\n"
    "n=n===B.hW||n===B.hX}else n=!1\n"
    "s=n?6:7\n"
    "break\n"
    "case 6:s=8\n"
    "return A.i(p.uD(o),$async$G8)\n"
    "case 8:s=1\n"
    "break\n"
    'case 7:p.fd(B.jb,"\\u672a\\u77e5\\u8bc4\\u4f30\\u7528\\u4f8b: "+a,a)'
)

NEW_BB3_BRANCH = (
    'break}s=a==="hello_then_stock_analysis"?3:a==="turn_plan_routing"?9:4\n'
    "break\n"
    "case 3:s=5\n"
    "return A.i(p.uC(),$async$G8)\n"
    "case 5:s=1\n"
    "break\n"
    "case 9:s=10\n"
    "return A.i(p.uF(a),$async$G8)\n"
    "case 10:s=1\n"
    "break\n"
    "case 4:o=A.it(p.ax,new A.arw(a))\n"
    "if(o!=null){n=o.f\n"
    "n=n===B.hW||n===B.hX}else n=!1\n"
    "s=n?6:7\n"
    "break\n"
    "case 6:s=8\n"
    "return A.i(p.uD(o),$async$G8)\n"
    "case 8:s=1\n"
    "break\n"
    'case 7:p.fd(B.jb,"\\u672a\\u77e5\\u8bc4\\u4f30\\u7528\\u4f8b: "+a,a)'
)

UF_METHOD = r"""uF(a){return this.aTQ(a)},
aTQ(b4){var s=0,r=A.u(t.H),q=1,p=[],o=[],n=this,m,l,k,j,i,h,g,f,e,d,c,b,a,a0,a1,a2
var $async$uF=A.p(function(b5,b6){if(b5===1){p.push(b6)
s=q}for(;;)switch(s){case 0:a=n.kd(b4)
m=new A.c1(Date.now(),0,!1)
l="run-"+m.a
k=A.bA_(b4,!1,null,null,null,l,null,n.a6Q().p2.gh(0),null,m,B.l9,a.b)
n.go=k
a0=n.ay
a0.hw(a0,0,k)
n.ch.sh(0,l)
s=2
return A.i(n.rn(k),$async$uF)
case 2:n.CW.sh(0,!0)
j=n.cx
i=j.Y$
i===$&&A.d()
J.dc(i,b4,new A.pi(B.l9,null,null))
n.fd(B.co,"======== \u5f00\u59cb TurnPlan \u8bc4\u4f30\uff08plan_only\uff09========",b4)
q=4
h=$.bEP()
g=t.f
f=t.z
e=A
s=5
return A.i(h.eB(),$async$uF)
case 5:s=6
return A.i(h.a.hh(A.df()+"/v1/dashboard/eval/cases/"+b4+"/run",null,A.cF(h.dh(),null,null),f),$async$uF)
case 6:d=e.ao(g.a(b6.a),t.N,f)
c=J.f(J.F(d,"ok"),!1)
b=J.F(d,"passed")
a=J.F(d,"total")
a1=J.F(d,"duration_ms")
n.fd(B.co,"TurnPlan: "+A.k(b)+"/"+A.k(a)+" passed, "+A.k(a1)+"ms",b4)
h=J.F(d,"results")
if(t.j.b(h)){for(g=J.aT(h);g.v();){a2=g.gO(g)
e=t.f.b(a2)?A.ao(a2,t.N,f):null
if(e!=null){a2=J.f(J.F(e,"passed"),!1)
n.fd(a2?B.co:B.jb,J.F(e,"message"),b4)}}}k=n.go
k.f=c?B.la:B.lb
k.e=new A.c1(Date.now(),0,!1)
k.y=J.f(a1,0)
if(!c){k.z=J.F(d,"status")
J.dc(j.Y$,b4,new A.pi(B.lb,k.y,k.z))}else{J.dc(j.Y$,b4,new A.pi(B.la,k.y,null))
n.fd(B.nE,"TurnPlan \u8bc4\u4f30\u901a\u8fc7\uff08"+B.h.W(k.y/1000,1)+"s\uff09",b4)}o.push(9)
s=8
break
case 4:q=3
a2=p.pop()
n.fd(B.jb,"TurnPlan \u8bc4\u4f30\u5931\u8d25: "+A.k(A.am(a2)),b4)
k=n.go
if(k!=null){k.f=B.lb
k.e=new A.c1(Date.now(),0,!1)
k.y=B.p.dK(new A.c1(Date.now(),0,!1).fe(m).a,1000)
k.z=J.v(A.am(a2))
J.dc(j.Y$,b4,new A.pi(B.lb,k.y,k.z))}o.push(9)
s=8
break
case 3:s=2
break
case 8:q=1
n.CW.sh(0,!1)
a1=j.gh(0)
i.r=a1
i.U(a1)
n.fd(B.co,"======== TurnPlan \u8bc4\u4f30\u7ed3\u675f ========",b4)
b=n.go
n.go=null
s=b!=null?10:11
break
case 10:s=12
return A.i(n.rn(b),$async$uF)
case 12:case 11:a=a0.a1$
a0=a0.gh(0)
a.r=a0
a.U(a0)
s=o.pop()
break
case 9:return A.r(null,r)
case 1:return A.q(p.at(-1),r)}})
return A.t($async$uF,r)},"""

UF_INSERT_AFTER = "return A.t($async$G8,r)},\nAY(a){return this.apA(a)},"


def patch_main_dart_js(content: str) -> str:
    if "aTQ(b4)" in content and 'a==="turn_plan_routing"?9' in content:
        return content
    if OLD_BB3_BRANCH not in content:
        raise RuntimeError("bb3 branch pattern not found in main.dart.js")
    content = content.replace(OLD_BB3_BRANCH, NEW_BB3_BRANCH, 1)
    if UF_INSERT_AFTER not in content:
        raise RuntimeError("uF insert anchor not found in main.dart.js")
    content = content.replace(
        UF_INSERT_AFTER,
        "return A.t($async$G8,r)},\n" + UF_METHOD + "\nAY(a){return this.apA(a)},",
        1,
    )
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
    remote_script = "/tmp/patch_trading_op_turnplan_eval.py"
    sftp.put(str(script_path), remote_script)
    sftp.close()

    ts = datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")
    ssh_run(client, f"cp {remote_js} {remote_js}.bak-{ts}")
    ssh_run(client, f"python3 {remote_script} --file {remote_js}")

    print("=== sync web -> nginx container ===")
    ssh_run(
        client,
        "NGINX=$(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'); "
        f"echo nginx=$NGINX; "
        f"docker cp {web_dir}/. ${{NGINX}}:/usr/share/nginx/html/; "
        "docker exec $NGINX sh -c 'grep -c \"turn_plan_routing\\\"?9\" /usr/share/nginx/html/main.dart.js; "
        "grep -c aTQ /usr/share/nginx/html/main.dart.js'",
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
