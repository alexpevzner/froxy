+++
title = "TProxy Configuration"
+++
<script src="/js/api.js" defer> </script>
<script src="/js/conf.js" defer> </script>

**Server Configuration**
<table >
	<tbody>
	<tr>
		<td>Server (host or host:port):</td>
		<td><input id="addr" type="text" style="width: 95%;"/></td>
	</tr>
	<tr>
		<td>Login:</td>
		<td><input id="login" type="text" style="width: 95%;" /></td>
	</tr>
	<tr>
		<td>Password:</td>
		<td><input id="password" type="text" style="width: 95%;" /></td>
		<td>&nbsp;<input id="usekey" type="checkbox"/>Use SSH key instead
	</tr>
	<tr>
		<td><input type="button" value="Ok" onclick="tproxy.Ui(SubmitServerParams)"/></td>
	</tr>
	</tbody>
</table>

---
**SSH Key Management**

***Key paramaters***
<table>
	<tbody>
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
	  <td>Comment (optional):</td>
	  <td><input id="key-comment" type="text"" /></td>
	</tr>
	<tr>
	  <td>SHA-256&nbsp;fingerprint:</td>
	  <td><div id="key-sha256">0000000000000000000000000000000000000000000000000000000000000000<div></td>
	</tr>
	<tr>
	  <td>MD5&nbsp;fingerprint:</td>
	  <td><div id="key-md5">00000000000000000000000000000000</div></td>
	</tr>
	<tr>
	  <td><input type="button" value="Generate" onclick="tproxy.Ui(GenKey)"/></td>
	  <td>Note, it will erase the previous content of this key, if any</td>
	</tr>
	<tbody>
</table>

**Public Key**
<br>
Add it into the the **authorized_keys** file at the server

<textarea id="key-pubtext" style="overflow:auto;resize:none" rows=4 cols=50 readonly>
</textarea>

<input type="button" value="Copy to Clipboard" onclick="tproxy.Ui(PubKeyCopy)"/>
<input type="button" value="Download As a File" onclick="tproxy.Ui(PubKeySave)"/>
