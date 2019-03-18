//
// Statistics counters page script
//

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
    var rq = tproxy.GetCounters();
    rq.OnSuccess = GetCountersCallback;
}

init();

// vim:ts=8:sw=2:et
