#!/usr/bin/env python3
"""Patch trading_operation main.dart.js so Dock Chat always parses clarify choices and renders a card."""
from __future__ import annotations

import argparse
import os
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path

AXZ_OLD = r"""axz(a){var s,r,q,p=a.i(0,"choices"),o=A.b([],t.s)
if(t.j.b(p))for(s=J.aW(p);s.v();){r=B.b.C(J.v(s.gO(s)))
if(r.length!==0)o.push(r)}s=a.i(0,"question")
q=s==null?null:B.b.C(J.v(s))
if(q==null)q=""
s=q.length===0
if(s&&o.length===0)return
s=s?"\u8bf7\u786e\u8ba4":q
this.fr.sh(0,new A.a5E(s,o))
s=this.CW
s.sh(0,s.gh(0)+1)}"""

AXZ_NEW = r"""axz(a){var s,r,q,p,n,m,l,k,o=A.b([],t.s)
if(a==null)return
n=a
m=a.i(0,"data")
if(m!=null&&t.f.b(m))n=A.al(m,t.N,t.z)
p=a.i(0,"choices")
if(p==null)p=a.i(0,"options")
if(p==null&&n!==a){p=n.i(0,"choices")
if(p==null)p=n.i(0,"options")}
if(p==null)p=a.i(0,"display_choices")
if(p==null&&n!==a)p=n.i(0,"display_choices")
if(p!=null&&!t.j.b(p)&&p.length!=null){l=A.b([],t.s)
for(s=0;s<p.length;++s)l.push(p[s])
p=l}if(t.j.b(p))for(s=J.aW(p);s.v();){r=s.gO(s)
if(t.f.b(r)){k=A.al(r,t.N,t.z)
q=k.i(0,"text")
if(q==null)q=k.i(0,"label")
r=q}r=r==null?"":B.b.C(J.v(r))
if(r.length!==0)o.push(r)}s=a.i(0,"question")
if(s==null&&n!==a)s=n.i(0,"question")
q=s==null?null:B.b.C(J.v(s))
if(q==null)q=""
s=q.length===0
if(s&&o.length===0)return
s=s?"\u8bf7\u786e\u8ba4":q
this.fr.sh(0,new A.a5E(s,o))
s=this.CW
s.sh(0,s.gh(0)+1)}"""

CP_U_OLD = r"""u(a){var s,r,q,p=this,o=null,n=p.f,m=n?8:10,l=A.x(8),k=A.ah(B.rO,1),j=p.c,i=t.p,h=A.b([A.U(A.b([B.a6D,B.bg,A.a0(A.l(j.a,o,o,o,o,A.af(o,o,B.ai,o,o,o,o,o,o,o,o,n?12:12.5,o,o,B.v,o,1.35,!0,o,o,o,o,o,o,o,o),o,o),1)],i),B.q,B.e,B.f,0,o,o),B.H],i)
if(j.b.length===0)h.push(B.aA1)
else{i=A.b([],i)
for(s=0;s<j.gE9().length;++s){r=p.aJQ(s,j.gE9()[s])
q=n?11.5:12
i.push(A.by3(o,B.k,A.l(r,o,o,o,o,new A.z(!0,o,o,o,o,o,q,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o),o,o),o,new A.ap8(p,s),B.Sz))}i.push(A.cG(B.aBs,o,o,o,p.e,o,A.ic(o,o,o,o,o,o,o,o,o,o,o,B.ab,B.cT,o,o,o,o,B.bv,o,o)))
h.push(A.by(B.I,i,B.a5,6,6))}return A.R(o,A.P(h,B.q,o,B.e,B.f,B.j),B.i,o,o,new A.L(B.e6,o,k,l,o,o,o,B.m),o,o,o,new A.W(0,m,0,0),B.bE,o,o,1/0)}"""

CP_U_NEW = r"""u(a){var s,r,q,p=this,o=null,n=p.f,m=n?14:18,l=A.x(16),k=A.ah(B.rO,2),j=p.c,i=t.p,h=A.b([A.U(A.b([B.a6D,B.bg,A.a0(A.l(j.a,o,o,o,o,A.af(o,o,B.ai,o,o,o,o,o,o,o,o,n?13:15,o,o,B.v,o,1.35,!0,o,o,o,o,o,o,o,o),o,o),1)],i),B.q,B.e,B.f,0,o,o),B.H],i)
if(j.b.length===0)h.push(B.aA1)
else{i=A.b([],i)
for(s=0;s<j.gE9().length;++s){r=p.aJQ(s,j.gE9()[s])
q=n?13:14
i.push(A.by3(o,B.k,A.l(r,o,o,o,o,new A.z(!0,o,o,o,o,o,q,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o,o),o,o),o,new A.ap8(p,s),B.SA))}i.push(A.cG(B.aBs,o,o,o,p.e,o,A.ic(o,o,o,o,o,o,o,o,o,o,o,B.ab,B.cT,o,o,o,o,B.bv,o,o)))
h.push(A.P(i,B.q,o,B.e,B.f,B.j))}return A.R(o,A.P(h,B.q,o,B.e,B.f,B.j),B.i,o,o,new A.L(B.e6,o,k,l,o,o,o,B.m),o,o,o,new A.W(12,m,12,10),B.bE,o,o,1/0)}"""


def patch_main_dart_js(content: str) -> str:
    if "ggClarifySheetV1" in content:
        return content
    if AXZ_OLD not in content:
        raise RuntimeError("axz clarify parser pattern not found in main.dart.js")
    if CP_U_OLD not in content:
        raise RuntimeError("Cp.u clarify card pattern not found in main.dart.js")
    patched = content.replace(AXZ_OLD, AXZ_NEW, 1).replace(CP_U_OLD, CP_U_NEW, 1)
    return patched.replace("A.a5E.prototype={", "A.a5E.prototype={ggClarifySheetV1(){return!0},", 1)


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
    remote_script = "/tmp/patch_trading_op_clarify_sheet.py"
    sftp.put(str(script_path), remote_script)
    sftp.close()

    ts = datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")
    ssh_run(client, f"cp {remote_js} {remote_js}.bak-clarify-{ts}")
    ssh_run(client, f"python3 {remote_script} --file {remote_js}")

    print("=== sync web -> nginx container ===")
    ssh_run(
        client,
        "NGINX=$(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'); "
        f"echo nginx=$NGINX; "
        f"docker cp {web_dir}/. ${{NGINX}}:/usr/share/nginx/html/; "
        'docker exec $NGINX sh -c \'grep -c ggClarifySheetV1 /usr/share/nginx/html/main.dart.js\'',
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
