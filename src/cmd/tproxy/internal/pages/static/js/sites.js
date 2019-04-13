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
    var elm = document.getElementById("add.host");
    if (elm.hasAttribute("hostname")) {
        var host = elm.getAttribute("hostname");
        var params = {
            host: host,
            rec: tproxy.UiGetInput("add.rec"),
            block: tproxy.UiGetInput("add.block")
        };

        tproxy.SetSite(params.host, params);

        tproxy.UiSetInput("add.host", "");
        tproxy.UiSetInput("add.rec", true);
        tproxy.UiSetInput("add.block", false);
        elm.removeAttribute("hostname");
    }
}

//
// Called when table button is clicked
//
function TableButtonClicked (button, rownum) {
    var row = table[rownum];
    var oldhost = row.getAttribute("host");

    switch (button) {
    case "update":
        var elm = document.getElementById(rownum + ".host");
        if (!elm.hasAttribute("hostname")) {
            break;
        }

        var params = {
            host: elm.getAttribute("hostname"),
            rec: tproxy.UiGetInput(rownum + ".rec"),
            block: tproxy.UiGetInput(rownum + ".block")
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
            tproxy.BgWatchStop(table.length + ".host");
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
        tproxy.BgWatch(n + ".host", "/api/domain", DomainChecked);
    }
}

//
// This function is called when domain name being edited
// by user was checked by Tproxy
//
function DomainChecked (id, reply) {
    var elm = document.getElementById(id);
    var ok = !!reply.host;

    if (ok) {
        elm.setAttribute("hostname", reply.data);
    } else {
        elm.removeAttribute("hostname");
    }

    if (elm.value && !ok) {
        elm.style.borderColor = "red";
        elm.style.borderStyle = "dashed";
    } else {
        elm.style.borderColor = "";
        elm.style.borderStyle = "";
    }
}

//
// Page initialization
//
function init () {
    tproxy.BgPoll("/api/sites", UpdateTable);
    tproxy.BgWatch("add.host", "/api/domain", DomainChecked);
}

init();

// vim:ts=8:sw=4:et
