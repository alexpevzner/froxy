//
// Site list page script
//

// ----- Static variables -----
//
// Array of table rows, one per site, grows or shrinks dynamically
//
var table = [];

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
// Resize table
//
function ResizeTable (sz) {
    if (table.length > sz) {
        while(table.length > sz) {
            table.pop().remove();
        }
    } else {
        var tbody = document.getElementById("tbody");

        while(table.length < sz) {
            var row = document.getElementById("template").cloneNode(true);
            row.hidden = false;

            inputs = row.getElementsByTagName("input");
            for (var i = 0; i < inputs.length; i ++) {
                var elm = inputs[i];
                var nm = elm.getAttribute("name");
                elm.id = table.length + "." + nm;
            }

            tbody.appendChild(row);
            table.push(row);
        }
    }
}

//
// Update table
//
function UpdateTable (sites) {
    rq = tproxy.GetSites();

    rq.OnSuccess = function (sites) {
        ResizeTable(sites.length);
        for (var row = 0; row < table.length; row ++) {
            tproxy.UiSetInput(row + ".host", sites[row].host);
        }
    };
}

//
// Page initialization
//
function init () {
    UpdateTable();
}

init();

// vim:ts=8:sw=4:et
