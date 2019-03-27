+++
title = "Statistics Counters"
+++
<script src="/js/api.js" defer> </script>
<script src="/js/counters.js" defer> </script>

Various statistics counters collected here for debugging and troubleshooting
purposes

Name                              | Value
----------------------------------|---------------
Direct TCP Connections            | <div id="tcp_conns"></div>
SSH Client Sessions               | <div id="ssh_sessions"></div>
SSH-tunneled TCP Connections      | <div id="ssh_conns"></div>
HTTP requests received            | <div id="http_rq_received"></div>
HTTP requests pending             | <div id="http_rq_pending"></div>
HTTP requests handled directly    | <div id="http_rq_direct"></div>
HTTP requests forwarded to server | <div id="http_rq_forwarded"></div>
HTTP requests blocked             | <div id="http_rq_blocked"></div>

