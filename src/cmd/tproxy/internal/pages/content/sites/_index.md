+++
title = "TProxy Sites"

# vim:ts=8:sw=2:et
+++
<script src="/js/api.js" defer> </script>
<script src="/js/sites.js" defer> </script>

Here you can edit list of sites which will be accessed via server.
In most cases you will want to check the "With subdomains" button

<fieldset><legend>Add new site</legend>
  <table >
    <tbody>
      <tr>
        <td><input id="add.host" type="text" style="width: 95%;" placeholder="Enter domain or url"/></td>
        <td>&nbsp;<input id="add.rec" type="checkbox" checked />With subdomains</td>
        <td>&nbsp;<input id="add.block" type="checkbox" />Block</td>
        <td><input type="button" value="Add" onclick="tproxy.Ui(AddSite)" /></td>
      </tr>
    </tbody>
  </table >
</fieldset>

<fieldset><legend>Manage existent sites</legend>
  <table>
    <tbody id="tbody">
      <tr id="template" hidden>
        <td><input name="host" type="text" style="width: 95%;" /></td>
        <td>&nbsp;<input name="rec" type="checkbox" checked /> With subdomains</td>
        <td>&nbsp;<input name="block" type="checkbox" />Block</td>
        <td><input name="update" type="button" value="Update"/></td>
        <td><input name="del" type="button" value="Del"/></td>
      </tr>
    </tbody>
  </table>
</fieldset>
