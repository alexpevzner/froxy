//
// JavaScript API for Froxy configuration pages
//

"use strict";

//
// All public symbols belong to the froxy namespace
// A nested froxy._ namespace is for internally used symbols
//
var froxy = {_: {}};

// ----- Internal functions. Don't use directly -----
//
// Write a debug message to JS console
//
froxy._.debug = console.log;

// ----- Internal variables. Don't use directly -----
//
// Init-on-demand flag
//
froxy._.init_done = false;

//
// Non-zero, when executing callback from input tag
//
froxy._.uiguard = 0;

//
// Count of active HTTP requests, generated as result of
// user interaction
//
froxy._.ui = 0;

//
// Count of active HTTP requests
//
froxy._.rq_count = 0;

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
froxy._.http_request = function(method, query, data) {
    froxy._.init();

    // Adjust query
    query = location.origin + query;

    // Log the event
    froxy._.debug(method, query);
    if (data) {
        froxy._.debug(data);
    }

    // Create a request
    var rq = {
        _xrq:      new XMLHttpRequest(),
        _ui:       froxy._.uiguard,
        canceled:  false,
        OnSuccess: function() {},
        OnError:   function() {}
    };

    rq._xrq.open(method, query, true);

    // Add a couple of methods
    rq.SetRequestHeader = function (name, value) {
        rq._xrq.setRequestHeader(name, value);
    };
    rq.GetResponseHeader = function (name) {
        return rq._xrq.getResponseHeader(name);
    };

    rq.Cancel = function () {
        rq._xrq.abort();
        rq.canceled = true;
        rq._finished();
    };

    // Request hooks
    rq._started = function () {
        froxy._.rq_count ++;
        if (rq._ui) {
            if (!froxy._.ui) {
                froxy._.uitimer = window.setTimeout(
                    froxy._.pleasewait.bind(null, true),
                    100
                );
            }
            froxy._.ui ++;
        }

    };

    rq._finished = function () {
        if (rq._ui) {
            froxy._.ui --;
            if (!froxy._.ui) {
                window.clearTimeout(froxy._.uitimer);
                froxy._.pleasewait(false);
            }
        }

        froxy._.rq_count --;
        froxy._.UiQueueRun();
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
                    err = froxy._.http_interror("JSON error: " + ex, query);
                }

                if (!err && body) {
                    if (body.data) {
                        data = body.data;
                    } else if (body.err && body.err.text) {
                        err = body.err;
                    } else {
                        err = froxy._.http_interror("Invalid responce from Froxy", query);
                    }
                }
            } else {
                if (rq._xrq.responseText) {
                    err = froxy._.http_interror(rq._xrq.responseText, query);
                } else if (rq._xrq.statusText) {
                    err = froxy._.http_interror(rq._xrq.statusText, query);
                } else {
                    err = froxy._.http_interror("HTTP request failed", query);
                }
            }

            if (err) {
                froxy._.debug(method, query, "err:", err);
                if (rq.OnError) {
                    rq.OnError(err);
                }
            } else {
                froxy._.debug(method, query, "data:", data);
                if (rq.OnSuccess) {
                    rq.OnSuccess(data);
                }
            }

            rq._finished();
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
        }, 0
    );

    rq._started();
    return rq;
};

//
// Create HTTP error object
//
froxy._.http_error = function(text, reason, object) {
    return {
        text:   text,
        reason: reason,
        object: object
    };
};

//
// Create HTTP error object with "Internal error" test
//
froxy._.http_interror = function(reason, object) {
    return froxy._.http_error("Internal error", reason, object);
};

// ----- UI helpers -----
//
// Display or remove "please wait" animation on a top of the current page
//
froxy._.pleasewait = function (enable) {
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
froxy.GetServerParams = function() {
    return froxy._.http_request("GET", "/api/server");
};

//
// Set server parameters - returns HTTP request
//
froxy.SetServerParams = function(addr, login, password, keyid) {
    var d = {
        addr: addr,
        login: login,
        password: password,
        keyid: keyid
    };
    return froxy._.http_request("PUT", "/api/server", d);
};

//
// Get list of sites - returns HTTP request
//
froxy.GetSites  = function() {
    return froxy._.http_request("GET", "/api/sites");
};

//
// Set a site parameters - returns HTTP request
//
froxy.SetSite  = function(host, params) {
    var q = "/api/sites";
    if (host) {
        q += "?" + encodeURIComponent(host);
    }

    return froxy._.http_request("PUT", q, params);
};

//
// Delete a site - returns HTTP request
//
froxy.DelSite  = function(host) {
    var q = "/api/sites?" + encodeURIComponent(host);
    return froxy._.http_request("DEL", q);
};

//
// Get statistics counters
//
froxy.GetCounters = function () {
    var q = "/api/counters";

    return froxy._.http_request("GET", q);
};

// ----- Key management -----
//
// Get all keys
//
froxy.GetKeys = function () {
    return froxy._.http_request("GET", "/api/keys");
};

//
// Generate key
//
froxy.GenKey = function (type, comment) {
    return froxy._.http_request(
        "POST",
        "/api/keys",
        { type: type, comment: comment }
    );
};

//
// Update key
//
froxy.UpdateKey = function (id, comment) {
    return froxy._.http_request(
        "PUT",
        "/api/keys?" + id,
        { comment: comment }
    );
};

//
// Delete key
//
froxy.DeleteKey = function (id) {
    return froxy._.http_request(
        "DEL",
        "/api/keys?" + id
    );
};

// ----- DOM helpers -----
//
// Get all (including indirect) children of a given element
//
froxy.DomChildren = function (element) {
    if (!element.children) {
        return [];
    }

    var fulllist = [];
    var children = [];
    var i;
    for (i = 0; i < element.children.length; i ++) {
        children.push(element.children[i]);
    }

    fulllist = fulllist.concat(children);
    for (i = 0; i < children.length; i ++) {
        fulllist = fulllist.concat(froxy.DomChildren(children[i]));
    }

    return fulllist;
};

// ----- UI helper functions -----
//
// Queue of Ui callbacks, differed until completion of HTTP
// requests being executed
//
froxy._.UiQueue = [];

//
// Enqueue Ui callback
//
froxy._.UiQueuePush = function (fn) {
    froxy._.uiguard ++;
    froxy._.UiQueue.push(fn);
};

//
// Execute queued Ui callbacks
//
froxy._.UiQueueRun = function () {
    while (froxy._.rq_count == 0 && froxy._.UiQueue.length) {
        var fn = froxy._.UiQueue.shift();
        fn();
        froxy._.uiguard --;
    }
};

//
// Wrapper for functions that called as input events handlers.
//
// Usage:
//     <input type="button" onclick="froxy.Ui(ButtonCallback)" />
//
// Wrapping input events handlers into this wrapper ensures proper
// synchronization between handling user actions and asynchronous
// execution of http requests initiated from such a handlers
//
froxy.Ui = function(fn) {
    if (froxy._.ui == 0) {
        froxy._.UiQueuePush(fn);
        froxy._.UiQueueRun();
    }
};

//
// Get value of particular control
//
froxy.UiGetInput = function(id) {
    var obj = document.getElementById(id);

    if (!obj) {
        return undefined;
    }

    switch (obj.tagName) {
    case "DIV":
    case "SPAN":
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
froxy.UiSetInput = function(id, value) {
    var obj = document.getElementById(id);

    if (!obj) {
        return;
    }

    if (value == undefined ) {
        value = "";
    }

    switch (obj.tagName) {
    case "DIV":
    case "SPAN":
    case "LEGEND":
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
froxy.UiSetStatus = function(color, text) {
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
froxy._.BgStartStatus = function () {
    var OnSuccess = function (state) {
        var color = "black";
        switch (state.state) {
        case "noconfig":    color = "olive"; break;
        case "trying":      color = "green"; break;
        case "established": color = "steelblue"; break;
        }

        froxy.UiSetStatus(color, state.info);
    };

    var OnError = function () {
        froxy.UiSetStatus("red", "Froxy not responding");
        froxy._.BgReloadWhenReady();
    };

    froxy.BgPoll("/api/state", OnSuccess, OnError);
};

//
// Monitor Froxy state and reload current page when it becomes ready
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
froxy._.BgReloadWhenReady = function() {
    setTimeout(function () {
        var rq = froxy._.http_request("GET", "/api/state");
        rq.OnSuccess = function () { location.reload(); };
        rq.OnError = froxy._.BgReloadWhenReady;
    }, 1000);
};

//
// Poll particular WebApi resource for change
//
froxy.BgPoll = function (url, OnSuccess, OnError) {
    froxy._.init();

    var poll = froxy._.poll[url];
    if (!poll) {
        poll = froxy._.poll[url] = {
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
froxy.BgWatch = function (id, url, callback) {
    var elm = document.getElementById(id);
    if (!elm) {
        return;
    }

    froxy.BgWatchStop(id);

    var watch = froxy._.watch[id] = {};

    elm.oninput = function () {
        if (watch.rq) {
            watch.textChanged = true;
        } else {
            watch.fire();
        }
    };

    // Fire the request to server
    watch.fire = function () {
        var q = url + "?" + froxy.UiGetInput(id);
        watch.rq = froxy._.http_request("GET", q);

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
// Stop watch initiated by froxy.BgWatch()
//
froxy.BgWatchStop = function (id) {
    var elm = document.getElementById(id);
    var watch = froxy._.watch[id];

    if (elm) {
        elm.onchange = function (){};
    }

    if (watch && watch.rq) {
        watch.rq.Cancel();
    }

    delete(froxy._.watch[id]);
};

//
// Initialize BgPoll
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
froxy._.BgPollInit = function () {
    froxy._.poll_url = "ws://" + location.host + "/api/poll";
    froxy._.poll_sock = new WebSocket(froxy._.poll_url);
    froxy._.poll_sock.onopen = froxy._.poll_sock_onopen;
    froxy._.poll_sock.onmessage = froxy._.poll_sock_onmessage;
    froxy._.poll_sock.onerror = froxy._.poll_sock_onerror;
    froxy._.poll_sock.onclose = froxy._.poll_sock_onclose;
    froxy._.poll = {};
    froxy._.watch = {};
};

//
// Poll websocket onopen callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
froxy._.poll_sock_onopen = function () {
    for (var path in froxy._.poll) {
        froxy._.poll_sock.send(JSON.stringify({path: path}));
    }
};

//
// Poll websocket onmessage callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
froxy._.poll_sock_onmessage = function (event) {
    // Parse received message
    var data;
    try {
        data = JSON.parse(event.data);
    } catch (ex) {
        froxy._.poll_sock_onerror("JSON error: " + ex);
        return;
    }

    var path = data.path;
    var tag = data.tag;
    var poll = froxy._.poll[path];

    if (!poll || !data.data.data) {
        return; // FIXME -- raise an error
    }

    // Notify subscribers
    var OnSuccess = poll.OnSuccess;
    for (var i = 0; i < OnSuccess.length; i ++) {
        OnSuccess[i](data.data.data);
    }

    // Reschedule poll
    froxy._.poll_sock.send(JSON.stringify({path: path, tag: tag}));
};

//
// Poll websocket onerror callback
//
// THIS IS INTERNAL FUNCTION, DON'T CALL IT DIRECTLY
//
froxy._.poll_sock_onerror = function (event) {
    froxy._.poll_sock.close();

    var err = froxy._.http_interror(event, froxy._.poll_url);

    for (var path in froxy._.poll) {
        var poll = froxy._.poll[path];
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
froxy._.poll_sock_onclose = function (event) {
    if (event.wasClean || !event.reason) {
        // Note, this event can be raised either on Froxy disconnect
        // or on window reload, and it is hard to distinguish between
        // these cases
        //
        // If we react immediately, in a case of page reload the
        // error status blinks for a moment before page reloaded.
        // It's not a big harm, but looks inaccurate, To fix this
        // cosmetic problem, we introduce a little delay here
        //
        // FIXME, this issue requires a better investigation
        setTimeout( function () {
            froxy._.poll_sock_onerror("websocked suddenly closed");
        }, 100);
    } else {
        froxy._.poll_sock_onerror(event.reason);
    }
};

// ----- Initialization -----
//
// Initialize stuff
//
froxy._.init = function() {
    if (!froxy._.init_done) {
        froxy._.init_done = true;
        froxy._.BgPollInit();
        froxy._.BgStartStatus();
    }
};

// vim:ts=8:sw=4:et
