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
// Count of active HTTP requests, generated as result of
// user interaction
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
        canceled:  false,
        OnSuccess: function() {},
        OnError:   function() {}
    };

    if (rq._ui) {
        if (!tproxy._.ui) {
            tproxy._.uitimer = window.setTimeout(
                tproxy._.pleasewait.bind(null, true),
                100
            );
        }
        tproxy._.ui ++;
    }

    rq._xrq.open(method, query, true);

    // Add a couple of methods
    rq.SetRequestHeader = function (name, value) {
        rq._xrq.setRequestHeader(name, value);
    };
    rq.GetResponseHeader = function (name) {
        return rq._xrq.getResponseHeader(name);
    };

    rq.Cancel = function() {
        rq._xrq.abort();
        rq.canceled = true;
    };

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
                if (!tproxy._.ui) {
                    window.clearTimeout(tproxy._.uitimer);
                    tproxy._.pleasewait(false);
                }
            }
        }
    };

    // Submit the request
    if (data) {
        data = JSON.stringify(data);
    }

    setTimeout(
        function() {
            if (!rq.canceled) {
                rq._xrq.send(data);
            }
        }, 0);

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

// ----- UI helpers -----
//
// Display or remove "please wait" animation on a top of the current page
//
tproxy._.pleasewait = function (enable) {
    var body = document.body;
    var wait = document.getElementById("id_wait");

    if (!body) {
        return;
    }

    if (enable) {
        if (wait) {
            return;
        }

        body.style.position = "relative";

        wait = document.createElement("div");
        wait.id = "id_wait";
        wait.className = "wait";
        wait.innerHTML = "<div class=\"ring\" id=\"id_ring\"></div>";

        body.appendChild(wait);
    } else {
        if (!wait) {
            return;
        }

        body.removeChild(wait);
    }
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
tproxy.SetServerParams = function(addr, login, password, keyid) {
    var d = {
        addr: addr,
        login: login,
        password: password,
        keyid: keyid
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
tproxy.GetCounters = function () {
    var q = "/api/counters";

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
tproxy.UpdateKey = function (id, comment) {
    return tproxy._.http_request(
        "PUT",
        "/api/keys?" + id,
        { comment: comment }
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
    case "DIV":
        return obj.innerText;

    case "INPUT":
        switch (obj.type) {
        case "text":
            return obj.value;

        case "checkbox":
            return !!obj.checked;
        }
        break;

    case "SELECT":
    case "TEXTAREA":
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
        obj.innerText = value;
        break;

    case "INPUT":
        switch (obj.type) {
        case "text":
            obj.value = value;
            break;

        case "checkbox":
            obj.checked = !!value;
            break;
        }
        break;

    case "SELECT":
    case "TEXTAREA":
        obj.value = value;
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
// Update status
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.BgStartStatus = function () {
    var OnSuccess = function (state) {
        var color = "black";
        switch (state.state) {
        case "noconfig":    color = "olive"; break;
        case "trying":      color = "green"; break;
        case "established": color = "steelblue"; break;
        }

        tproxy.UiSetStatus(color, state.info);
    };

    var OnError = function () {
        tproxy.UiSetStatus("red", "TProxy not responding");
        tproxy._.BgReloadWhenReady();
    };

    tproxy.BgPoll("/api/state", OnSuccess, OnError);
};

//
// Monitor TProxy state and reload current page when it becomes ready
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.BgReloadWhenReady = function() {
    var rq = tproxy._.http_request("GET", "/api/state");
    rq.OnSuccess = function () {
        location.reload();
    };
    rq.OnError = function () {
        setTimeout(tproxy._.BgReloadWhenReady, 1000);
    };
};

//
// Poll particular WebApi resource for change
//
tproxy.BgPoll = function (url, OnSuccess, OnError) {
    var poll = tproxy._.poll[url];
    if (!poll) {
        poll = tproxy._.poll[url] = {
            OnSuccess: [],
            OnError: []
        };
    }

    if (OnSuccess) {
        poll.OnSuccess.push(OnSuccess);
    }

    if (OnError) {
        poll.OnError.push(OnError);
    }
};

//
// Watch particular text input control for changes
//
//   id                - id of the text input control
//   url               - webapi URL (i.e., "/api/domain")
//   callback(id,data) - user callback
//
// When user changes text, contained by the specified
// text input control, webapi GET request to the specified
// url will be scheduled, giving current text as a
// parameter, and when reply will be received,
// the callback will be invoked with reply data
// used as a callback parameter
//
// This function is useful for autocompletion,
// syntax checking during editing and similar tasks
//
tproxy.BgWatch = function (id, url, callback) {
    var elm = document.getElementById(id);
    if (!elm) {
        return;
    }

    tproxy.BgWatchStop(id);

    var watch = tproxy._.watch[id] = {};

    elm.oninput = function () {
        if (watch.rq) {
            watch.textChanged = true;
        } else {
            watch.fire();
        }
    };

    // Fire the request to server
    watch.fire = function () {
        var q = url + "?" + tproxy.UiGetInput(id);
        watch.rq = tproxy._.http_request("GET", q);

        watch.rq.OnSuccess = function (data) {
            delete(watch.rq);
            if (watch.textChanged) {
                watch.fire();
            }
            watch.textChanged = false;
            callback(id, data);
        };
    };

    watch.fire();
};

//
// Stop watch initiated by tproxy.BgWatch()
//
tproxy.BgWatchStop = function (id) {
    var elm = document.getElementById(id);
    var watch = tproxy._.watch[id];

    if (elm) {
        elm.onchange = function (){};
    }

    if (watch && watch.rq) {
        watch.rq.Cancel();
    }

    delete(tproxy._.watch[id]);
};

//
// Initialize BgPoll
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.BgPollInit = function () {
    tproxy._.poll_url = "ws://" + location.host + "/api/poll";
    tproxy._.poll_sock = new WebSocket(tproxy._.poll_url);
    tproxy._.poll_sock.onopen = tproxy._.poll_sock_onopen;
    tproxy._.poll_sock.onmessage = tproxy._.poll_sock_onmessage;
    tproxy._.poll_sock.onerror = tproxy._.poll_sock_onerror;
    tproxy._.poll_sock.onclose = tproxy._.poll_sock_onclose;
    tproxy._.poll = {};
    tproxy._.watch = {};
};

//
// Poll websocket onopen callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.poll_sock_onopen = function () {
    for (var path in tproxy._.poll) {
        tproxy._.poll_sock.send(JSON.stringify({path: path}));
    }
};

//
// Poll websocket onmessage callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.poll_sock_onmessage = function (event) {
    // Parse received message
    var data;
    try {
        data = JSON.parse(event.data);
    } catch (ex) {
        tproxy._.poll_sock_onerror("JSON error: " + ex);
        return;
    }

    var path = data.path;
    var tag = data.tag;
    var poll = tproxy._.poll[path];

    if (!poll || !data.data.data) {
        return; // FIXME -- raise an error
    }

    // Notify subscribers
    var OnSuccess = poll.OnSuccess;
    for (var i = 0; i < OnSuccess.length; i ++) {
        OnSuccess[i](data.data.data);
    }

    // Reschedule poll
    tproxy._.poll_sock.send(JSON.stringify({path: path, tag: tag}));
};

//
// Poll websocket onerror callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.poll_sock_onerror = function (event) {
    tproxy._.poll_sock.close();

    var err = tproxy._.http_interror(event, tproxy._.poll_url);

    for (var path in tproxy._.poll) {
        var poll = tproxy._.poll[path];
        var OnError = poll.OnError;
        for (var i = 0; i < OnError.length; i ++) {
            OnError[i](err);
        }
    }
};

//
// Poll websocket onclose callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
tproxy._.poll_sock_onclose = function (event) {
    if (event.wasClean || !event.reason) {
        tproxy._.poll_sock_onerror("websocked suddenly closed");
    } else {
        tproxy._.poll_sock_onerror(event.reason);
    }
};

// ----- Initialization -----
//
// Initialize stuff
//
tproxy._.init = function() {
    // Preload /css/tproxy.css
    var head  = document.getElementsByTagName("head")[0];
    var link  = document.createElement("link");
    link.rel  = "stylesheet";
    link.type = "text/css";
    link.href = "/css/tproxy.css";
    link.media = "all";
    head.appendChild(link);

    tproxy._.BgPollInit();
    tproxy._.BgStartStatus();
};

tproxy._.init();

// vim:ts=8:sw=4:et
