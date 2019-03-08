//
// JavaScript API for TProxy configuration pages
//

//
// All public symbols belong to the tprpxy namespacea
// A nested tprpxy._ namespace is for internally used symbols
//
var tprpxy = {_: {}};

// ----- Internal functions. Don't use directly -----
//
// Write a debug message to JS console
//
tprpxy._.debug = console.log;

// ----- HTTP requests handling -----
//
// Create asynchronous HTTP request
//
tprpxy._.http_request = function(method, query, data) {
    // Adjust query
    query = location.origin + "/api" + query;

    // Log the event
    tprpxy._.debug(method, query);
    if (data) {
        tprpxy._.debug(data);
    }

    // Create a request
    var rq = {
        _xrq:      new XMLHttpRequest(),
        onSuccess: function() {},
        onError:   function() {}
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
                    err = tprpxy._.http_error("JSON error: " + ex, query);
                }

                if (!err && body) {
                    if (body.data) {
                        data = body.data;
                    } else if (body.err && body.err.text) {
                        err = body.err;
                    } else {
                        err = tprpxy._.http_error("Invalid server response received", query);
                    }
                }
            } else {
                if (rq._xrq.responseText) {
                    err = tprpxy._.http_error(rq._xrq.responseText, query);
                } else if (rq._xrq.statusText) {
                    err = tprpxy._.http_error(rq._xrq.statusText, query);
                } else {
                    err = tprpxy._.http_error("HTTP request failed", query);
                }
            }

            if (err) {
                if (rq.onError) {
                    rq.onError(err);
                }
            } else {
                if (rq.onSuccess) {
                    rq.onSuccess(data);
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
tprpxy._.http_error = function(reason, object) {
    return {
        text:   "Internal error",
        reason: reason,
        object: object
    };
};

// vim:ts=8:sw=2:et
