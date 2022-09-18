+++
title = "Froxy Configuration"

# vim:ts=8:sw=4:et
+++
<script src="/js/api.js" defer> </script>
<script src="/js/conf.js" defer> </script>

<fieldset><legend>Server Configuration</legend>
<table >
    <tbody>
    <tr>
        <td>Server (host or host:port):</td>
        <td><input id="addr" type="text" onkeydown="froxy.UiClickOnEnter('ok',event)"/></td>
    </tr>
    <tr>
        <td>Login:</td>
        <td><input id="login" type="text" onkeydown="froxy.UiClickOnEnter('ok',event)"/></td>
    </tr>
    <tr>
        <td>Authentication method:</td>
        <td>
            <select id="auth" onchange="AuthMethodOnChange()">
                <option value="auth.none">-- Please, choose --</option>
                <option value="auth.password">Password</option>
            </select>
        </td>
    </tr>
    <tr>
        <td>Password:</td>
        <td><input id="password" type="text" disabled onkeydown="froxy.UiClickOnEnter('ok',event)"/></td>
    </tr>
    <tr>
        <td><input id="ok" type="button" value="Ok" onclick="froxy.Ui(SubmitServerParams)"/></td>
    </tr>
    </tbody>
</table>
</fieldset>
