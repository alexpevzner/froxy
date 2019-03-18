+++
title = "TProxy Sites"
+++
<script src="/api.js" defer> </script>
<script src="/sites.js" defer> </script>

Here you can edit list of sites which will be accessed via server.
In most cases you will want to check the "With subdomains" button

<table >
	<tbody id="tbody">
	<tr>
		<td><input id="add.host" type="text" style="width: 95%;"/></td>
		<td>&nbsp;<input id="add.rec" type="checkbox" checked />With subdomains</td>
		<td>&nbsp;<input id="add.block" type="checkbox" />Block</td>
		<td><input type="button" value="Add" onclick="AddSite()" /></td>
	</tr>
	<tr id="template" hidden>
		<td><input name="host" type="text" style="width: 95%;" /></td>
		<td>&nbsp;<input name="rec" type="checkbox" checked /> With subdomains</td>
		<td>&nbsp;<input name="block" type="checkbox" />Block</td>
		<td><input name="update" type="button" value="Update"/></td>
		<td><input name="del" type="button" value="Del"/></td>
	</tr>
	</tbody>
</table>

