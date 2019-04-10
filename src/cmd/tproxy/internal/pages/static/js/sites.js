//
// Site list page script
//

"use strict";

// ----- Static variables -----
//
// Array of table rows, one per site, grows or shrinks dynamically
//
var table = [];

//
// Add a site
//
function AddSite () {
    var params = {
        host: tproxy.UiGetInput("add.host"),
        rec: tproxy.UiGetInput("add.rec"),
        block: tproxy.UiGetInput("add.block")
    };

    if (params.host) {
        tproxy.SetSite(params.host, params);

        tproxy.UiSetInput("add.host", "");
        tproxy.UiSetInput("add.rec", true);
        tproxy.UiSetInput("add.block", false);
    }
}

//
// Called when table button is clicked
//
function TableButtonClicked (button, rownum) {
    console.log("click", button, rownum);
    var row = table[rownum];
    var oldhost = row.getAttribute("host");

    switch (button) {
    case "update":
        var params = {
            host: tproxy.UiGetInput(rownum + "." + "host"),
            rec: tproxy.UiGetInput(rownum + "." + "rec"),
            block: tproxy.UiGetInput(rownum + "." + "block")
        };

        tproxy.SetSite(oldhost, params);
        break;

    case "del":
        tproxy.DelSite(oldhost);
        break;
    }
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
                        return tproxy.Ui.bind(null, function() {
                            TableButtonClicked(n, i);
                        });
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
        tproxy.UiSetInput(n + ".block", sites[n].block);
        table[n].setAttribute("host", sites[n].host);
    }
}

//
// Page initialization
//
function init () {
    tproxy.BgPoll("/api/sites", UpdateTable);
}

init();

// vim:ts=8:sw=4:et
