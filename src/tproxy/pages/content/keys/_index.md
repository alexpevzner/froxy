+++
title = "SSH Keys Management"
+++
<script src="/js/api.js" defer> </script>
<script src="/js/keys.js" defer> </script>

**Need more keys?**
<table>
    <tbody>
        <tr>
            <td>Comment&nbsp;(optional):</td>
            <td><input id="key-comment" type="text"" /></td>
        </tr>
        <tr>
            <td>Key Type:</td>
            <td>
                <select id="key-type">
                    <option value="rsa-2048">RSA-2048</option>
                    <option value="rsa-4096" selected="true">RSA-4096</option>
                    <option value="ecdsa-256">ECDSA-256</option>
                    <option value="ecdsa-384">ECDSA-384</option>
                    <option value="ecdsa-521">ECDSA-521</option>
                    <option value="ed25519">Ed25519</option>
                </select>
            </td>
        </tr>
        <tr>
            <td><input type="button" value="Generate" onclick="tproxy.Ui(GenKey)"/></td>
        </tr>
    <tbody>
</table>

**Manage keys you have:**
<table>
    <tbody>
        <tr><td>
            <table>
                <tbody>
                    <tr>
                        <td colspan=2>
                            <input type="checkbox"/>
                            Enable this key
                            <input type="button" value="Delete this Key"/>
                        </td>
                    </tr>
                    <tr>
                        <td>Comment:</td><td><div id="list.comment"/></td>
                    </tr>
                    <tr>
                        <td>Key Type:</td><div id="list.type"/></td>
                    </tr>
                    <tr>
                        <td>SHA-256&nbsp;fingerprint:</td>
                        <td><div id="list.sha256">0000000000000000000000000000000000000000000000000000000000000000<div></td>
                    </tr>
                    <tr>
                        <td>MD5&nbsp;fingerprint:</td>
                        <td><div id="list.md5">00000000000000000000000000000000</div></td>
                    </tr>
                    <tr>
                        <td colspan=2>**Public Key**</td>
                    </tr>
                    <tr>
                        <td colspan=2>Add it into the the **authorized_keys** file at the server</td>
                    </tr>
                    <tr>
                        <td colspan=2>
                            <textarea id="list.pubtext" style="overflow:auto;resize:none" rows=4 cols=50 readonly></textarea>
                        </td>
                    </tr>
                    <tr>
                        <td colspan=2>
                            <input id="list.pub-copy" type="button" value="Copy to Clipboard" onclick="tproxy.Ui(PubKeyCopy)"/>
                            <input id="list.pub-save" type="button" value="Download As a File" onclick="tproxy.Ui(PubKeySave)"/>
                        </td>
                    </tr>
                </tbody>
            </table>
        </td></tr>
    </tbody>
</table>

[comment]: # (vim:ts=8:sw=4:et)
