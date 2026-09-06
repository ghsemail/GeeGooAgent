/* Dock Chat clarify overlay: Flutter canvas often never mounts the option sheet. */
(function () {
  if (window.__ggClarifyOverlayV2) return;
  window.__ggClarifyOverlayV2 = true;

  var style = document.createElement("style");
  style.textContent = [
    "#gg-clarify-root{position:fixed;left:0;right:0;bottom:0;z-index:2147483000;pointer-events:none;font-family:ui-sans-serif,system-ui,sans-serif;}",
    "#gg-clarify-card{pointer-events:auto;margin:0 auto 88px;max-width:560px;background:#fff8ed;border:2px solid #fdba74;border-radius:16px;box-shadow:0 12px 40px rgba(0,0,0,.18);padding:16px 18px 14px;color:#1c1917;}",
    "#gg-clarify-q{font-size:16px;font-weight:700;margin:0 0 12px;line-height:1.4;}",
    "#gg-clarify-ops{display:flex;flex-direction:column;gap:8px;}",
    ".gg-clarify-btn{appearance:none;border:1px solid #fdba74;background:#fff;border-radius:12px;padding:10px 12px;text-align:left;font-size:14px;cursor:pointer;line-height:1.35;}",
    ".gg-clarify-btn:hover{background:#ffedd5;}",
    ".gg-clarify-btn b{color:#c2410c;margin-right:8px;}",
    "#gg-clarify-skip{margin-top:8px;background:transparent;border:none;color:#78716c;cursor:pointer;font-size:12px;}"
  ].join("");
  document.documentElement.appendChild(style);

  var root = null;
  var lastSession = "";
  var lastHeaders = {};
  var lastClarifyUrl = "/op_agent/v1/chat/clarify";

  function hide() {
    if (root && root.parentNode) root.parentNode.removeChild(root);
    root = null;
  }

  function headerObj(h) {
    var out = {};
    if (!h) return out;
    if (typeof h.forEach === "function") {
      h.forEach(function (v, k) { out[k] = v; });
      return out;
    }
    Object.keys(h).forEach(function (k) { out[k] = h[k]; });
    return out;
  }

  function postAnswer(answer, skip) {
    var headers = Object.assign({ "Content-Type": "application/json" }, lastHeaders);
    fetch(lastClarifyUrl, {
      method: "POST",
      headers: headers,
      body: JSON.stringify({ session_id: lastSession, answer: answer || "", skip: !!skip })
    }).catch(function () {});
    hide();
  }

  function show(sessionId, question, choices, streamUrl, headers) {
    lastSession = sessionId || lastSession;
    lastHeaders = headerObj(headers);
    if (streamUrl) lastClarifyUrl = String(streamUrl).replace(/\/v1\/chat\/stream.*$/, "/v1/chat/clarify");
    var opts = (choices || []).map(function (c) {
      if (typeof c === "string") return c;
      if (c && typeof c === "object") return c.text || c.label || "";
      return String(c || "");
    }).filter(Boolean);
    if (!opts.length) return;
    hide();
    root = document.createElement("div");
    root.id = "gg-clarify-root";
    var card = document.createElement("div");
    card.id = "gg-clarify-card";
    var q = document.createElement("div");
    q.id = "gg-clarify-q";
    q.textContent = question || "请选择";
    var ops = document.createElement("div");
    ops.id = "gg-clarify-ops";
    opts.forEach(function (text, i) {
      var b = document.createElement("button");
      b.className = "gg-clarify-btn";
      b.type = "button";
      var lab = String.fromCharCode(65 + i);
      b.innerHTML = "<b>" + lab + "</b>" + text.replace(/[&<>]/g, function (ch) {
        return ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[ch];
      });
      b.addEventListener("click", function () { postAnswer(text, false); });
      ops.appendChild(b);
    });
    var skip = document.createElement("button");
    skip.id = "gg-clarify-skip";
    skip.type = "button";
    skip.textContent = "跳过";
    skip.addEventListener("click", function () { postAnswer("", true); });
    card.appendChild(q);
    card.appendChild(ops);
    card.appendChild(skip);
    root.appendChild(card);
    document.body.appendChild(root);
  }

  function extractChoices(obj) {
    if (!obj || typeof obj !== "object") return [];
    var nested = obj.data && typeof obj.data === "object" ? obj.data : {};
    var raw = obj.choices || obj.options || nested.choices || nested.options || obj.display_choices || nested.display_choices;
    if (!raw) return [];
    if (Array.isArray(raw)) return raw;
    return [];
  }

  function handlePayload(eventName, obj, streamUrl, headers) {
    if (!obj || typeof obj !== "object") return;
    if (obj.session_id) lastSession = obj.session_id;
    if (obj.data && obj.data.session_id) lastSession = obj.data.session_id;
    var name = String(eventName || obj.event || "").replace(/\r/g, "").trim();
    if (name === "turn_end" || name === "done") {
      hide();
      return;
    }
    var clarify = name === "clarify" || obj.item_type === "clarify_prompt" || (obj.data && obj.data.item_type === "clarify_prompt");
    if (!clarify) return;
    var q = obj.question || (obj.data && obj.data.question) || "";
    show(lastSession, q, extractChoices(obj), streamUrl, headers);
  }

  async function consumeSSE(stream, streamUrl, headers) {
    var reader = stream.getReader();
    var dec = new TextDecoder();
    var buf = "";
    try {
      for (;;) {
        var chunk = await reader.read();
        if (chunk.done) break;
        buf += dec.decode(chunk.value, { stream: true });
        buf = buf.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
        var idx;
        while ((idx = buf.indexOf("\n\n")) >= 0) {
          var raw = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          var ev = "message";
          var data = "";
          raw.split("\n").forEach(function (line) {
            if (line.indexOf("event:") === 0) ev = line.slice(6).trim();
            else if (line.indexOf("data:") === 0) data = line.slice(5).trim();
          });
          if (!data) continue;
          var obj;
          try { obj = JSON.parse(data); } catch (e) { continue; }
          handlePayload(ev, obj, streamUrl, headers);
        }
      }
    } catch (e) {}
  }

  var orig = window.fetch;
  window.fetch = function (input, init) {
    var url = typeof input === "string" ? input : (input && input.url) || "";
    var p = orig.apply(this, arguments);
    if (String(url).indexOf("/v1/chat/stream") === -1) return p;
    return p.then(function (res) {
      if (!res || !res.body || !res.ok || typeof res.body.tee !== "function") return res;
      var parts = res.body.tee();
      consumeSSE(parts[0], String(url), (init && init.headers) || (input && input.headers));
      return new Response(parts[1], res);
    });
  };
})();
