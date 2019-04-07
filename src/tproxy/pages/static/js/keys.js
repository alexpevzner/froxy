//
// SSH Keys management
//

"use strict";

// ----- Static variables -----
//
// Table of keys, one row per key, grows or shrinks dynamically
//
var table = [];

//
// Reset "generate key" form parameters
//
function ResetGenKeyParameters () {
    tproxy.UiSetInput("key-type", "rsa-4096");
    tproxy.UiSetInput("key-comment", "");
}

//
// Generate SSH key
//
function GenKey () {
    var rq = tproxy.GenKey(
        tproxy.UiGetInput("key-type"),
        tproxy.UiGetInput("key-comment"),
        true
    );

    rq.OnSuccess = function () {
        ResetGenKeyParameters();
        ReloadTable();
    };
}

//
// Delete the key
//
function DeleteKey (keyid) {
    var ok = confirm(
        "Deleted keys cannot be recovered\n" +
        "Are you sure you want to continue?"
    );
    if (!ok) {
        return;
    }

    var rq = tproxy.DeleteKey(keyid);
    rq.OnSuccess = ReloadTable; 
}

//
// Copy public key to clipboard
//
function PubKeyCopy (row) {
    var elm = document.getElementById(row + ".pubkey");

    if (elm && elm.value) {
        elm.select();
        document.execCommand("copy");
    }
}

//
// Save public key to file
//
function PubKeySave (row) {
    var pubkey = tproxy.UiGetInput(row + ".pubkey");
    var keytype = tproxy.UiGetInput(row + ".type");

    if (pubkey) {
        var a = document.createElement("a");
        var type = keytype ? keytype.split("-")[0] : "";
        var filename = type ? "id_" + type + ".pub": "id.pub";

        a.setAttribute("href", "data:text/plain;charset=utf-8," + encodeURIComponent(pubkey));
        a.setAttribute("download", filename);

        a.style.display = "none";
        document.body.appendChild(a);

        a.click();

        document.body.removeChild(a);
    }
}

//
// Handle user input from keys table controls
//
// Parameters:
//   input  - input name ("enable", "delete" etc)
//   elm    - HTML element event related to
//   row    - row number
//
function TableInputAction (input, elm, row) {
    var keyid = table[row].getAttribute("keyid");

    switch (input) {
    case "delete":
        DeleteKey(keyid);
        break;

    case "enable":
    case "sendcomment":
        tproxy.UpdateKey(
            keyid,
            tproxy.UiGetInput(row + ".enable"),
            tproxy.UiGetInput(row + ".comment")
        );
        break;

    case "pub-copy":
        PubKeyCopy(row);
        break;

    case "pub-save":
        PubKeySave(row);
        break;
    }
}

//
// Update table of existent keys
//
function UpdateKeys (keys) {
    var sz = keys.length;


    // Update "you have no keys" notice
    var nokeys = document.getElementById("nokeys");
    if (nokeys) {
        nokeys.hidden = !!sz;
    }

    // Sort keys
    keys.sort(function (a, b) {
        return a.date.localeCompare(b.date);
    });

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

            var children = tproxy.DomChildren(row);
            for (var i = 0; i < children.length; i ++) {
                var elm = children[i];
                var id = elm.id;
                if (id && id.startsWith("add.")) {
                    elm.id = table.length + id.slice(3);
                    if (elm.tagName == "INPUT") {
                        var f = TableInputAction.bind(
                            null,
                            id.slice(4),
                            elm,
                            table.length
                        );
                        f = tproxy.Ui.bind(null, f);

                        switch (elm.type) {
                        case "text":
                            elm.oninput = f;
                            break;
                        case "checkbox":
                        case "button":
                            elm.onclick = f;
                            break;
                        }
                    }
                }
            }

            tbody.appendChild(row);
            table.push(row);
        }
    }

    // Update rows
    for (var n = 0; n < table.length; n ++) {
        tproxy.UiSetInput(n + ".enable", keys[n].enabled);
        tproxy.UiSetInput(n + ".comment", keys[n].comment);
        tproxy.UiSetInput(n + ".type", keys[n].type);
        tproxy.UiSetInput(n + ".sha256", keys[n].fp_sha256);
        tproxy.UiSetInput(n + ".md5", keys[n].fp_md5);
        tproxy.UiSetInput(n + ".pubkey", keys[n].pubkey);
        table[n].setAttribute("keyid", keys[n].id);

        var months = [
            "Jan", "Feb", "Mar", "Apr", "May", "Jun",
            "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"
        ];

        function fmtNum(n) {
            return (n < 10 ? "0" : "") + n;
        }

        var date = new Date(keys[n].date);
        var stddate =
            date.getDate() + " " +
            months[date.getMonth()] + " " +
            date.getFullYear() + " " +
            fmtNum(date.getHours()) + ":" +
            fmtNum(date.getMinutes()) + "." +
            fmtNum(date.getSeconds());

        tproxy.UiSetInput(n + ".ctime", stddate);

        elm = document.getElementById(n + ".hr");
        if (elm) {
            elm.hidden = n == (table.length - 1);
        }
    }
}

//
// Reload table of keys
//
function ReloadTable () {
    var rq = tproxy.GetKeys();
    rq.OnSuccess = UpdateKeys;
}

// ----- Initialization -----
//
// Page initialization
//
function init() {
    ResetGenKeyParameters();
    ReloadTable();
}


init();

// vim:ts=8:sw=2:et
