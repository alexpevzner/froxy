//
// Statistics counters page script
//

"use strict";

//
// tproxy.GetCounters callback
//
function GetCountersCallback (data) {
    // Resubmit a request
    var rq = tproxy.GetCounters(data.tag.toString());
    rq.OnSuccess = GetCountersCallback;

    // Update page
    for (var name in data) {
        if (data.hasOwnProperty(name)) {
            var c = document.getElementById(name);
            if (c) {
                c.innerHTML = data[name];
            }
        }
    }
}

//
// Page initialization
//
function init() {
    tproxy.BgPoll("/api/counters", GetCountersCallback);
}

init();

// vim:ts=8:sw=2:et
