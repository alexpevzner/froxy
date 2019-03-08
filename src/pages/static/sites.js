//
// Site list page script
//

//
// Add a site
//
function AddSite () {
    var host = tproxy.UiGetInput("add.host");
    var rec = tproxy.UiGetInput("add.rec");

    if (host) {
        tproxy.SetSite(host, host, rec);
        tproxy.UiSetInput("add.host", "");
        tproxy.UiSetInput("add.rec", true);
    }
}

//
// Page initialization
//
function init () {
}

init();

// vim:ts=8:sw=2:et
