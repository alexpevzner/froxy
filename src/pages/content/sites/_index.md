+++
title = "TProxy Sites"
+++
<script src="/api.js"defer> </script>
<script src="/sites.js"defer> </script>

Here you can edit list of sites which will be accessed via server.
In most cases you will want to check the "With subdomains" button

<table >
	<tbody>
	<tr>
		<td><input id="add.host" type="text" style="width: 95%;"/></td>
		<td>&nbsp;<input id="add.rec" type="checkbox" checked /> &nbsp;With subdomains
		<td><input type="button" value="Add" onclick="AddSite()" /></td>
	</tr>
	<tr>
		<td><input type="text" style="width: 95%;" disabled/></td>
		<td>&nbsp;<input type="checkbox" checked disabled /> &nbsp;With subdomains
		<td><input type="button" value="Del"/></td>
	</tr>
	</tbody>
</table>

