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
        var rq = tproxy.SetSite(host, host, rec);

        rq.OnSuccess = ReloadTable;
        tproxy.UiSetInput("add.host", "");
        tproxy.UiSetInput("add.rec", true);
    }
}

//
// Called when table button is clicked
//
function TableButtonClicked (button, rownum) {
    console.log("click", button, rownum);
    var row = table[rownum];
    var oldhost = row.getAttribute("host");
    var rq;

    switch (button) {
    case "update":
        var host = tproxy.UiGetInput(rownum + "." + "host");
        var rec = tproxy.UiGetInput(rownum + "." + "rec");

        rq = tproxy.SetSite(oldhost, host, rec);
        break;

    case "del":
        rq = tproxy.DelSite(oldhost);
        break;
    }

    rq.OnSuccess = ReloadTable;
}

//
// Update table of sites
//
function UpdateTable (sites) {
    var sz = sites.length;

    // Sort sites
    sites.sort(function(a, b) { return a.host.localeCompare(b.host); });

    // Resize table
    if (table.length > sz) {
        while(table.length > sz) {
            table.pop().remove();
        }
    } else {
        var tbody = document.getElementById("tbody");

        while(table.length < sz) {
            var row = document.getElementById("template").cloneNode(true);

            row.hidden = false;

            var inputs = row.getElementsByTagName("input");
            for (var i = 0; i < inputs.length; i ++) {
                var elm = inputs[i];
                var nm = elm.getAttribute("name");

                elm.id = table.length + "." + nm;

                if (elm.type == "button") {
                    elm.onclick = function(n, i) {
                        return function() {
                            TableButtonClicked(n, i);
                        };
                    }(nm, table.length);
                }
            }

            tbody.appendChild(row);
            table.push(row);
        }
    }

    // Update rows
    for (var n = 0; n < table.length; n ++) {
        tproxy.UiSetInput(n + ".host", sites[n].host);
        tproxy.UiSetInput(n + ".rec", sites[n].rec);
        table[n].setAttribute("host", sites[n].host);
    }
}

//
// Reload table of sites
//
function ReloadTable () {
    var rq = tproxy.GetSites();
    rq.OnSuccess = UpdateTable;
}

//
// Page initialization
//
function init () {
    ReloadTable();
}

init();

// vim:ts=8:sw=4:et
