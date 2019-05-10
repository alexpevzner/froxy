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
    froxy.UiSetInput("key-type", "ecdsa-384");
    froxy.UiSetInput("key-comment", "");
}

//
// Generate SSH key
//
function GenKey () {
    var rq = froxy.GenKey(
        froxy.UiGetInput("key-type"),
        froxy.UiGetInput("key-comment")
    );

    rq.OnSuccess = function () {
        ResetGenKeyParameters();
    };
}

//
// Delete the key
//
function DeleteKey (keyid) {
    froxy.DeleteKey(keyid);
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
    var pubkey = froxy.UiGetInput(row + ".pubkey");
    var keytype = froxy.UiGetInput(row + ".type");

    if (!pubkey) {
        return;
    }

    var filename = type ? "id_" + type + ".pub": "id.pub";

    if (navigator.msSaveOrOpenBlob) {
        navigator.msSaveOrOpenBlob(
            new Blob([pubkey], { type: "application/x-pem-file" }),
            filename
        );
    } else {
        var a = document.createElement("a");
        var type = keytype ? keytype.split("-")[0] : "";

        a.setAttribute("href", "data:application/x-pem-file," + encodeURIComponent(pubkey));
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
//   input  - input name ("delete", "confirm-delete" etc)
//   elm    - HTML element event related to
//   row    - row number
//
function TableInputAction (input, elm, row) {
    var keyid = table[row].getAttribute("keyid");

    switch (input) {
    case "delete":
        var confirm = document.getElementById(row + ".confirm-delete");
        confirm.hidden = !froxy.UiGetInput(row + ".delete");
        break;

    case "confirm-delete":
        DeleteKey(keyid);
        break;

    case "sendcomment":
        froxy.UpdateKey(
            keyid,
            froxy.UiGetInput(row + ".comment")
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
    var row;

    // Update "you have no keys" notice
    var nokeys = document.getElementById("nokeys");
    if (nokeys) {
        nokeys.hidden = !!sz;
    }

    // Resize table
    if (table.length > sz) {
        while(table.length > sz) {
            row = table.pop();
            row.parentNode.removeChild(row);
        }
    } else {
        var tbody = document.getElementById("tbody");

        while(table.length < sz) {
            row = document.getElementById("template").cloneNode(true);

            row.hidden = false;

            var children = froxy.DomChildren(row);
            for (var i = 0; i < children.length; i ++) {
                var elm = children[i];
                var id = elm.id;
                if (id && (id.substring(0,4) == "add.")) {
                    elm.id = table.length + id.slice(3);
                    if (elm.tagName == "INPUT") {
                        var f = TableInputAction.bind(
                            null,
                            id.slice(4),
                            elm,
                            table.length
                        );
                        f = froxy.Ui.bind(null, f);

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
        froxy.UiSetInput(n + ".keytag", "Key " + (n + 1));
        froxy.UiSetInput(n + ".delete", false);
        document.getElementById(n + ".confirm-delete").hidden = true;
        froxy.UiSetInput(n + ".comment", keys[n].comment);

        var type = keys[n].type;
        if (type == "ed25519") {
            type = "Ed25519";
        } else {
            type = type.toUpperCase();
        }
        froxy.UiSetInput(n + ".type", type);

        froxy.UiSetInput(n + ".sha256", keys[n].fp_sha256);
        froxy.UiSetInput(n + ".md5", keys[n].fp_md5);
        froxy.UiSetInput(n + ".pubkey", keys[n].pubkey);

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

        froxy.UiSetInput(n + ".ctime", stddate);
    }
}

// ----- Initialization -----
//
// Page initialization
//
function init() {
    ResetGenKeyParameters();
    froxy.BgPoll("/api/keys", UpdateKeys);
}


window.onload = init;

// vim:ts=8:sw=2:et
