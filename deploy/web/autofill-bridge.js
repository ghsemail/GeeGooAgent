/* Chrome password autofill bridge for Flutter Web (trading_operation).
 *
 * Flutter renders transparent <input> elements without login autofill hints.
 * This script patches username/password fields and keeps a hidden <form> in sync
 * so Chrome can recognize and fill saved credentials.
 */
(function () {
  if (window.__ggAutofillBridgeV1) return;
  window.__ggAutofillBridgeV1 = true;

  var FORM_ID = "gg-autofill-form";
  var USER_ID = "gg-autofill-username";
  var PASS_ID = "gg-autofill-password";

  function isFlutterInput(el) {
    if (!el || el.tagName !== "INPUT") return false;
    if (el.id === USER_ID || el.id === PASS_ID) return false;
    var cls = el.className || "";
    return cls.indexOf("flt-") >= 0 || el.getAttribute("data-semantics-role") != null;
  }

  function visibleInputs() {
    return Array.prototype.slice
      .call(document.querySelectorAll("input"))
      .filter(function (el) {
        if (el.id === USER_ID || el.id === PASS_ID) return false;
        if (el.type === "hidden" || el.type === "checkbox" || el.type === "radio") return false;
        return isFlutterInput(el) || el.type === "password" || el.type === "text";
      });
  }

  function findLoginPair(inputs) {
    var password = null;
    var username = null;
    for (var i = 0; i < inputs.length; i++) {
      if (inputs[i].type === "password") password = inputs[i];
    }
    if (!password) return null;

    for (var j = 0; j < inputs.length; j++) {
      var candidate = inputs[j];
      if (candidate === password) continue;
      if (candidate.type === "text" || candidate.type === "" || candidate.type === "email") {
        username = candidate;
        break;
      }
    }
    return { username: username, password: password };
  }

  function ensureForm() {
    var form = document.getElementById(FORM_ID);
    if (form) return form;

    form = document.createElement("form");
    form.id = FORM_ID;
    form.method = "post";
    form.action = "/op_catalog/login";
    form.autocomplete = "on";
    form.setAttribute("aria-hidden", "true");
    form.style.cssText =
      "position:fixed;left:-9999px;top:0;width:1px;height:1px;opacity:0;pointer-events:none;overflow:hidden;";

    var user = document.createElement("input");
    user.type = "text";
    user.name = "username";
    user.id = USER_ID;
    user.autocomplete = "username";
    user.tabIndex = -1;

    var pass = document.createElement("input");
    pass.type = "password";
    pass.name = "password";
    pass.id = PASS_ID;
    pass.autocomplete = "current-password";
    pass.tabIndex = -1;

    form.appendChild(user);
    form.appendChild(pass);

    var mount = document.body || document.documentElement;
    mount.appendChild(form);
    return form;
  }

  function bindMirror(source, mirror) {
    if (!source || source.__ggAutofillBound) return;
    source.__ggAutofillBound = true;

    function sync() {
      mirror.value = source.value || "";
    }

    source.addEventListener("input", sync);
    source.addEventListener("change", sync);
    source.addEventListener("blur", sync);
    sync();
  }

  function patchPair(pair) {
    if (!pair || !pair.password) return;

    ensureForm();
    var mirrorUser = document.getElementById(USER_ID);
    var mirrorPass = document.getElementById(PASS_ID);

    pair.password.autocomplete = "current-password";
    pair.password.name = "password";
    pair.password.setAttribute("autocapitalize", "off");
    pair.password.setAttribute("autocorrect", "off");

    bindMirror(pair.password, mirrorPass);

    if (pair.username) {
      pair.username.autocomplete = "username";
      pair.username.name = "username";
      pair.username.setAttribute("autocapitalize", "none");
      pair.username.setAttribute("autocorrect", "off");
      pair.username.setAttribute("spellcheck", "false");
      bindMirror(pair.username, mirrorUser);
    }
  }

  function scan() {
    var pair = findLoginPair(visibleInputs());
    if (pair) patchPair(pair);
  }

  var observer = new MutationObserver(function () {
    scan();
  });
  observer.observe(document.documentElement, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ["type", "class", "style"],
  });

  document.addEventListener("focusin", scan, true);
  window.addEventListener("load", scan);
  scan();
  setInterval(scan, 1500);
})();
