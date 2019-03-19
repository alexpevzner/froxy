+++
title = "TProxy Configuration"
+++
<script src="/api.js" defer> </script>
<script src="/conf.js" defer> </script>

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
		</td>
	</tr>
	<tr>
		<td><input type="button" value="Ok" onclick="tproxy.Ui(SubmitServerParams)"/></td>
	</tr>
	</tbody>
</table>

