//
// JavaScript API for TProxy configuration pages
//

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
        OnSuccess: function() {},
        OnError:   function() {}
    };


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
tproxy.SetSite  = function(oldhost, newhost, recursive) {
    var q = "/api/sites";
    if (oldhost) {
        q += "?" + encodeURIComponent(oldhost);
    }

    var d = {
        host: newhost, rec: recursive
    };

    return tproxy._.http_request("PUT", q, d);
};

//
// Delete a site - returns HTTP request
//
tproxy.DelSite  = function(oldhost) {
    var q = "/api/sites?" + encodeURIComponent(oldhost);
    return tproxy._.http_request("DEL", q);
};

// ----- UI helper functions -----
//
// Get value of particular control
//
tproxy.UiGetInput = function(id) {
    var obj = document.getElementById(id);
    if (obj && obj.tagName == "INPUT") {
        switch (obj.type) {
        case "text":
            return obj.value;

        case "checkbox":
            return !!obj.checked;
        }
    }
    return undefined;
};

//
// Get value of particular control
//
tproxy.UiSetInput = function(id, value) {
    var obj = document.getElementById(id);
    if (obj && obj.tagName == "INPUT") {
        switch (obj.type) {
        case "text":
            obj.value = value ? value : "";
            break;

        case "checkbox":
            obj.checked = !!value;
            break;
        }
    }
};

// vim:ts=8:sw=4:et
