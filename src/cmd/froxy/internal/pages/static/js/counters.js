//
// Statistics counters page script
//

"use strict";

//
// Poll Counters callback
//
function GetCountersCallback (data) {
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
    froxy.BgPoll("/api/counters", GetCountersCallback);
}

init();

// vim:ts=8:sw=2:et
