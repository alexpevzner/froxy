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
<div id="nokeys">You don't have any key...</div>
<table>
    <tbody id="tbody">
        <tr id="template" hidden><td>
            <table>
                <tbody>
                    <tr>
                        <td>
                            <input id="add.enable" type="checkbox"/>
                            Enable this key
                        </td>
                        <td>
                            <input id="add.delete" type="button" value="Delete this Key"/>
                        </td>
                    </tr>
                    <tr>
                        <td>
			    Comment:</td><td><input id="add.comment" type="text""/>
                            <input id="add.sendcomment" type="button" value="Update"/>
                        </td>
                    </tr>
                    <tr>
                        <td>Key Type:</td><td><div id="add.type"/></td>
                    </tr>
                    <tr>
                        <td>SHA-256&nbsp;fingerprint:</td>
                        <td><div id="add.sha256"/></td>
                    </tr>
                    <tr>
                        <td>MD5&nbsp;fingerprint:</td>
                        <td><div id="add.md5"/></td>
                    </tr>
                    <tr>
                        <td colspan=2>**Public Key**</td>
                    </tr>
                    <tr>
                        <td colspan=2>Add it into the the **authorized_keys** file at the server</td>
                    </tr>
                    <tr>
                        <td colspan=2>
                            <textarea id="add.pubkey" style="overflow:auto;resize:none" rows=4 cols=50 readonly></textarea>
                        </td>
                    </tr>
                    <tr>
                        <td colspan=2>
                            <input id="add.pub-copy" type="button" value="Copy to Clipboard"/>
                            <input id="add.pub-save" type="button" value="Download As a File"/>
                        </td>
                    </tr>
                </tbody>
            </table>
        </td></tr>
    </tbody>
</table>

[comment]: # (vim:ts=8:sw=4:et)
