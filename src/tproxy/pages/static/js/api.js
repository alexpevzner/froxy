//
// JavaScript API for TProxy configuration pages
//

"use strict";

//
// All public symbols belong to the tproxy namespacea
// A nested tproxy._ namespace is for internally used symbols
//
var tproxy = {_: {}};

// ----- Internal functions. Don't use directly -----
//
// Write a debug message to JS console
//
tproxy._.debug = console.log;

// ----- Internal variables. Don't use directly -----
//
// Non-zero, when executing callback from input tag
//
tproxy._.uiguard = 0;

//
// Count of active HTTP requests, associated with input tag callbacs
//
tproxy._.ui = 0;

// ----- HTTP requests handling -----
//
// Create asynchronous HTTP request
//
// Request properties, that can be set by caller:
//
//   function OnSuccess  - called on successful completion with
//                         received data as parameter. Received
//                         JSON is decoded into the JavaScript
//                         object
//
//   function OnError    - called on erroneous completion with
//                         error object as parameter. Error object
//                         is a JS object similar to following:
//                            {
//                               text:   "Internal error",
//                               reason: "HTTP request failed",
//                               object: "http://localhost:8888/api/sites"
//                            }
//
tproxy._.http_request = function(method, query, data) {
    // Adjust query
    query = location.origin + query;

    // Log the event
    tproxy._.debug(method, query);
    if (data) {
        tproxy._.debug(data);
    }

    // Create a request
    var rq = {
        _xrq:      new XMLHttpRequest(),
        _ui:       tproxy._.uiguard,
        OnSuccess: function() {},
        OnError:   function() {}
    };

    if (rq._ui) {
        tproxy._.ui ++;
    }

    rq._xrq.open(method, query, true);

    // Setup event handling
    rq._xrq.onreadystatechange = function () {
        if (rq._xrq.readyState == 4) {
            var data, err;

            if (rq._xrq.status == 200) {
                var body;

                try {
                    if (rq._xrq.responseText) {
                        body = JSON.parse(rq._xrq.responseText);
                    }
                } catch (ex) {
                    err = tproxy._.http_interror("JSON error: " + ex, query);
                }

                if (!err && body) {
                    if (body.data) {
                        data = body.data;
                    } else if (body.err && body.err.text) {
                        err = body.err;
                    } else {
                        err = tproxy._.http_interror("Invalid responce from TProxy", query);
                    }
                }
            } else {
                if (rq._xrq.responseText) {
                    err = tproxy._.http_interror(rq._xrq.responseText, query);
                } else if (rq._xrq.statusText) {
                    err = tproxy._.http_interror(rq._xrq.statusText, query);
                } else {
                    err = tproxy._.http_interror("HTTP request failed", query);
                }
            }

            if (err) {
                tproxy._.debug(method, query, "err:", err);
                if (rq.OnError) {
                    rq.OnError(err);
                }
            } else {
                tproxy._.debug(method, query, "data:", data);
                if (rq.OnSuccess) {
                    rq.OnSuccess(data);
                }
            }

            if (rq._ui) {
                tproxy._.ui --;
            }
        }
    };

    // Submit the request
    if (data) {
        rq._xrq.send(JSON.stringify(data));
    } else {
        rq._xrq.send();
    }

    return rq;
};

//
// Create HTTP error object
//
tproxy._.http_error = function(text, reason, object) {
    return {
        text:   text,
        reason: reason,
        object: object
    };
};

//
// Create HTTP error object with "Internal error" test
//
tproxy._.http_interror = function(reason, object) {
    return tproxy._.http_error("Internal error", reason, object);
};

// ----- Public API -----
//
// Get server parameters - returns HTTP request
//
tproxy.GetServerParams = function() {
    return tproxy._.http_request("GET", "/api/server");
};

//
// Set server parameters - returns HTTP request
//
tproxy.SetServerParams = function(addr, login, password) {
    var d = {
        addr: addr, login: login, password: password
    };
    return tproxy._.http_request("PUT", "/api/server", d);
};

//
// Get list of sites - returns HTTP request
//
tproxy.GetSites  = function() {
    return tproxy._.http_request("GET", "/api/sites");
};

//
// Set a site parameters - returns HTTP request
//
tproxy.SetSite  = function(host, params) {
    var q = "/api/sites";
    if (host) {
        q += "?" + encodeURIComponent(host);
    }

    return tproxy._.http_request("PUT", q, params);
};

//
// Delete a site - returns HTTP request
//
tproxy.DelSite  = function(host) {
    var q = "/api/sites?" + encodeURIComponent(host);
    return tproxy._.http_request("DEL", q);
};

//
// Get statistics counters
//
// tag, if defined, is a last known counters tag. If it is provided,
// request will block until counters change
//
tproxy.GetCounters = function (tag) {
    var q = "/api/counters";

    if (tag) {
        q += "?" + tag;
    }

    return tproxy._.http_request("GET", q);
};

// ----- Key management -----
//
// Get all keys
//
tproxy.GetKeys = function () {
    return tproxy._.http_request("GET", "/api/keys");
};

//
// Generate key
//
tproxy.GenKey = function (type, comment) {
    return tproxy._.http_request(
        "POST",
        "/api/keys",
        { type: type, comment: comment }
    );
};

//
// Update key
//
tproxy.UpdateKey = function (id, enabled, comment) {
    return tproxy._.http_request(
        "PUT",
        "/api/keys?" + id,
        { enabled: enabled, comment: comment }
    );
};

//
// Delete key
//
tproxy.DeleteKey = function (id) {
    return tproxy._.http_request(
        "DEL",
        "/api/keys?" + id
    );
};

// ----- DOM helpers -----
//
// Get all (including indirect) children of a given element
//
tproxy.DomChildren = function (element) {
    if (!element.children) {
        return [];
    }

    var children = Array.from(element.children);
    var fulllist = [];

    fulllist = fulllist.concat(children);
    for (var i = 0; i < children.length; i ++) {
        fulllist = fulllist.concat(tproxy.DomChildren(children[i]));
    }

    return fulllist;
};

// ----- UI helper functions -----
//
// Wrapper for functions that called as input events handlers.
//
// Usage:
//     <input type="button" onclick="tproxy.Ui(ButtonCallback)" />
//
// Wrapping input events handlers into this wrapper ensures proper
// synchronization between handling user actions and asynchronous
// execution of http requests initiated from such a handlers
//
tproxy.Ui = function(fn) {
    if (tproxy._.ui == 0) {
        tproxy._.uiguard ++;
        fn();
        tproxy._.uiguard --;
    }
};

//
// Get value of particular control
//
tproxy.UiGetInput = function(id) {
    var obj = document.getElementById(id);

    if (!obj) {
        return undefined;
    }

    switch (obj.tagName) {
    case "INPUT":
        switch (obj.type) {
        case "text":
            return obj.value;

        case "checkbox":
            return !!obj.checked;
        }
        break;

    case "SELECT":
        return obj.value;
    }

    return undefined;
};

//
// Get value of particular control
//
tproxy.UiSetInput = function(id, value) {
    var obj = document.getElementById(id);

    if (!obj) {
        return;
    }

    if (value == undefined ) {
        value = "";
    }

    switch (obj.tagName) {
    case "DIV":
    case "TEXTAREA":
        obj.innerText = value;
        break;

    case "INPUT":
        switch (obj.type) {
        case "text":
            obj.value = value ? value : "";
            break;

        case "checkbox":
            obj.checked = !!value;
            break;
        }
        break;
    }
};

//
// Set status string
//
tproxy.UiSetStatus = function(color, text) {
    var status = document.getElementById("status");
    if (!status) {
        return;
    }

    status.style.color = color;
    status.innerHTML = text;
};

// ----- Background activities -----
//
// Start status monitoring
//
tproxy.BgStartStatus = function(laststate) {
    var q = "/api/state";
    if (laststate) {
        q += "?" + laststate;
    }

    var rq = tproxy._.http_request("GET", q);
    rq.OnSuccess = function (state) {
        var color = "black";
        switch (state.state) {
        case "noconfig":    color = "olive"; break;
        case "trying":      color = "green"; break;
        case "established": color = "steelblue"; break;
        }

        tproxy.UiSetStatus(color, state.info);
        tproxy.BgStartStatus(state.state);
    };

    rq.OnError = function () {
        tproxy.UiSetStatus("red", "TProxy not responding");
        tproxy.BgReloadWhenReady();
    };
};

//
// Monitor TProxy state and reload current page when it becomes ready
//
tproxy.BgReloadWhenReady = function() {
    var rq = tproxy._.http_request("GET", "/api/state");
    rq.OnSuccess = function () {
        location.reload();
    };
    rq.OnError = function () {
        setTimeout(tproxy.BgReloadWhenReady, 1000);
    };
};

// ----- Initialization -----
//
// Initialize stuff
//
tproxy._.init = function() {
    tproxy.BgStartStatus();
};

tproxy._.init();

// vim:ts=8:sw=4:et
