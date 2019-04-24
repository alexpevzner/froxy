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
            <td><input id="key-comment" type="text" /></td>
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
            <td><input type="button" value="Generate" onclick="froxy.Ui(GenKey)"/></td>
        </tr>
    </tbody>
</table>

**Manage keys you have:**

<div id="nokeys">You don't have any key...</div>

<table>
    <tbody id="tbody">
        <tr id="template" hidden><td>
	    <fieldset><legend id="add.keytag"></legend>
		<table>
		    <tbody>
			<tr>
			    <td colspan="3">
				<div style="overflow:hidden;height:0em">
				    <textarea style="height:0;border:none;border-spacing:0" rows="1" cols="72" readonly></textarea>
				</div>
			    </td>
			</tr>
			<tr>
			    <td><div style="padding-right:3em">Created:</div></td>
			    <td><div id="add.ctime"></div>></td>
			</tr>
			<tr>
			    <td>Comment:</td>
			    <td>
			        <input id="add.comment" type="text"/>
				<input id="add.sendcomment" type="button" value="Update"/>
			    </td>
			</tr>
			<tr>
			    <td>Key Type:</td>
			    <td><div id="add.type"></div></td>
			</tr>
		    </tbody>
		</table>
		    <table>
			<tbody>
			<tr>
			    <td>
				<details>
				    <summary><strong>Fingerprints</strong></summary>
					<table>
					    <tbody>
						<tr>
						    <td>SHA-256</td><td>
						    <div id="add.sha256"></div></td>
						</tr>
						<tr>
						    <td>MD5</td><td>
						    <div id="add.md5"></div></td>
						</tr>
					    </tbody>
					</table>
				</details>
			    </td>
			</tr>
			<tr>
			    <td>
				<details>
				    <summary><strong>Public Key</strong></summary>
				    <table>
					<tbody>
					    <tr><td>
						Add it into the the <strong>$HOME/.ssh/authorized_keys</strong> file at the server:
					    </td></tr>
					    <tr><td>
						<textarea id="add.pubkey" style="overflow:auto;resize:none" rows="3" cols="70" readonly></textarea>
					    </td></tr>
					    <tr><td>
						<input id="add.pub-copy" type="button" value="Copy to Clipboard"/>
						<input id="add.pub-save" type="button" value="Download As a File"/>
					    </td></tr>
					</tbody>
				    </table>
				</details>
			    </td>
			</tr>
			<tr>
			    <td>
				<input id="add.delete" type="checkbox"/>
				Delete this Key
				<input id="add.confirm-delete" type="button" value="Confirm Delete" hidden/>
			    </td>
			</tr>
		    </tbody>
		</table>
	    </fieldset>
        </td></tr>
    </tbody>
</table>

[comment]: # (vim:ts=8:sw=4:et)
