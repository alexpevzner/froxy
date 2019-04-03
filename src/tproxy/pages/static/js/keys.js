//
// SSH Keys management
//

"use strict";

//
// Generate SSH key
//
function GenKey() {
    alert("Not implemented");
}

//
// Copy public key to clipboard
//
function PubKeyCopy() {
    var el = document.getElementById("key-pubtext");
    var text = el.value;

    if (el.value) {
        el.select();
        document.execCommand("copy");
    }
}

//
// Save public key to file
//
function PubKeySave() {
    alert("Not implemented");
}

// ----- Initialization -----
//
// Page initialization
//
function init() {
}


init();

// vim:ts=8:sw=2:et
