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
		<td><input id="addr" type="text"/></td>
	</tr>
	<tr>
		<td>Login:</td>
		<td><input id="login" type="text"/></td>
	</tr>
	<tr>
		<td>Password:</td>
		<td><input id="password" type="text" disabled/></td>
		<td><div id="password.comment">Not used if SSH keys are enabled</div></td>
	</tr>
	<tr>
		<td>Use SSH keys</td>
		<td><input id="usekey" type="checkbox" disabled onclick="tproxy.Ui(UseKeysClicked)"/>
		<td><div id="usekey.comment">[Check your SSH keys configuration](/keys)</div>
	</tr>
	<tr>
		<td><input type="button" value="Ok" onclick="tproxy.Ui(SubmitServerParams)"/></td>
	</tr>
	</tbody>
</table>
