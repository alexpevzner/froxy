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
            rec: froxy.UiGetInput("add.rec"),
            block: froxy.UiGetInput("add.block")
        };

        froxy.SetSite(params.host, params);

        froxy.UiSetInput("add.host", "");
        froxy.UiSetInput("add.rec", true);
        froxy.UiSetInput("add.block", false);
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
            rec: froxy.UiGetInput(rownum + ".rec"),
            block: froxy.UiGetInput(rownum + ".block")
        };

        froxy.SetSite(oldhost, params);
        break;

    case "del":
        froxy.DelSite(oldhost);
        break;
    }
}

//
// Update table of sites
//
function UpdateTable (sites) {
    var sz = sites.length;
    var row;

    // Sort sites
    sites.sort(function(a, b) { return a.host.localeCompare(b.host); });

    // Resize table
    if (table.length > sz) {
        while(table.length > sz) {
            row = table.pop();
            row.parentNode.removeChild(row);
            froxy.BgWatchStop(table.length + ".host");
        }
    } else {
        var tbody = document.getElementById("tbody");

        while(table.length < sz) {
            row = document.getElementById("template").cloneNode(true);

            row.hidden = false;

            var inputs = row.getElementsByTagName("input");
            for (var i = 0; i < inputs.length; i ++) {
                var elm = inputs[i];
                var nm = elm.getAttribute("name");

                elm.id = table.length + "." + nm;

                if (elm.type == "button") {
                    elm.onclick = function(n, i) {
                        return froxy.Ui.bind(null, function() {
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
        froxy.UiSetInput(n + ".host", sites[n].host);
        froxy.UiSetInput(n + ".rec", sites[n].rec);
        froxy.UiSetInput(n + ".block", sites[n].block);
        table[n].setAttribute("host", sites[n].host);
        froxy.BgWatch(n + ".host", "/api/domain", DomainChecked);
    }
}

//
// This function is called when domain name being edited
// by user was checked by Froxy
//
function DomainChecked (id, reply) {
    var elm = document.getElementById(id);
    var ok = !!reply.host;

    if (ok) {
        elm.setAttribute("hostname", reply.host);
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
    froxy.BgPoll("/api/sites", UpdateTable);
    froxy.BgWatch("add.host", "/api/domain", DomainChecked);
}

window.onload = init;

// vim:ts=8:sw=4:et
