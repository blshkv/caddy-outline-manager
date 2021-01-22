package outline

import "html/template"

var serverPanelTemplate = template.Must(template.New("").Parse(`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
<html>

<head profile="http://www.ietf.org/rfc/rfc2731.txt">

<meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1">
<meta http-equiv="Content-Language" content="en">
<meta http-equiv="Content-Style-Type" content="text/css">
<meta http-equiv="Content-Script-Type" content="text/javascript">

<title>Outline Manager</title>

<style>
table {
  font-family: arial, sans-serif;
  border-collapse: collapse;
  width: 100%;
}
td, th {
  border: 1px solid #dddddd;
  text-align: left;
  padding: 8px;
}
tr:nth-child(odd) {
  background-color: #dddddd;
}
</style>

</head>

<body onload = "JavaScript:auto_fresh(5000);">

<h2 id="outline-title">Outline Manager - {{ .Server.Total }} - <button type="button" onclick="add_user();">ADD USER</button><button type="button" id="button-refresh" onclick="set_refresh();">REFRESH ON</button><button type="button" onclick="exit();">EXIT</button></h2>

<table>
  <tr>
    <th>ID</th>
    <th>Name</th>
    <th>Expire Date</th>
    <th>Access URL</th>
    <th>Transferred</th>
    <th>Data Limit</th>
    <th>Client IP</th>
    <th>Online</th>
    <th>Enabled</th>
    <th>Days Left</th>
    <th></th>
  </tr>
  {{ range .Users }}
  <tr>
    <td>{{ .ID }}</td>
    <td>
      <input id="name-{{ .ID }}" value="{{ .Name }}" size="5" onkeydown="if(event.keyCode==13){rename_user({{ .JSID }});return false}"/>
    </td>
    <td>{{ .Expire }}</td>
    <td>
      <input type="text" value="{{ .AccessURL }}" id="url-{{ .ID }}" size="50"/>
      <button type="button" onclick="copy_ss_url({{ .JSID }});">COPY</button>
    </td>
    <td>{{ .TransferredBytes }}</td>
    <td>
      <input id="data-{{ .ID }}" value="{{ .Limit }}" size="4" onkeydown="if(event.keyCode==13){set_data_limit({{ .JSID }});return false}"/>GB
    </td>
    <td>{{ .IP }}</td>
    <td bgcolor="{{ .OnColor }}">{{ .Online }}</td>
    <td bgcolor="{{ .EnColor }}">{{ .Enabled }}<button type="button" onclick="change_user_status({{ .JSID }})">SWITCH</button></td>
    <td>
      <input id="time-{{ .ID }}" value="{{ .DaysLeft }}" size="2" onkeydown="if(event.keyCode==13){set_deadline({{ .JSID }});return false}"/>
    </td>
    <td>
      <button type="button" onclick="delete_user({{ .JSID }});">DELETE</button>
    </td>
  </tr>
  {{ end }}
</table>

<p>Admin Settings: Username: <input id="username" value="" size="10"/>  Password: <input id="password" type="password" value="" size="10"/><button type="button" onclick="set_manager();">MODIFY</button></p>

<script>
function add_user() {
  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  xmlHttp.open("POST", document.URL+"/user", false);
  xmlHttp.send(null);
}
</script>

<script>
function rename_user(id) {
  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  var url = document.URL+"/name?id="+id+"&name="+document.getElementById("name-"+id).value
  xmlHttp.open("PUT", url, false);
  xmlHttp.send(null);
}
</script>

<script>
function copy_ss_url(id) {
  var copyText = document.getElementById("url-"+id);

  copyText.select(); 
  copyText.setSelectionRange(0, 99999);

  document.execCommand("copy");

  alert("Copyed URL: " + copyText.value);
}
</script>

<script>
function set_data_limit(id) {
  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  var url = document.URL+"/data?id="+id+"&allowance="+document.getElementById("data-"+id).value
  xmlHttp.open("PUT", url, false);
  xmlHttp.send(null);
}
</script>

<script>
function change_user_status(id) {
  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  var url = document.URL+"/status?id="+id
  xmlHttp.open("PATCH", url, false);
  xmlHttp.send(null);
}
</script>

<script>
function delete_user(id) {
  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  var url = document.URL+"/id?id="+id
  xmlHttp.open("DELETE", url, false);
  xmlHttp.send(null);
}
</script>

<script>
function set_deadline(id) {
  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  var url = document.URL+"/deadline?id="+id+"&days="+document.getElementById("time-"+id).value
  xmlHttp.open("PUT", url, false);
  xmlHttp.send(null);
}
</script>

<script>
function close_current_window() {
  alert("Close");
  open(location, '_self').close();
}
</script>

<script>
function hide_title() {
  var ele = document.getElementById("outline-title");
  ele.innerText = "Outline Manager"
}
</script>

<script>
function set_manager() {
  var user = document.getElementById("username").value;
  var pass = document.getElementById("password").value;

  var xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
    setTimeout("location.reload();", 1000);
  }
  var url = document.URL+"/set/admin?user="+user+"&pass="+pass
  xmlHttp.open("POST", url, false);
  xmlHttp.send(null);
}
</script>

<script type="text/JavaScript">
function getCookie(cname) {
  var name = cname + "=";
  var decodedCookie = decodeURIComponent(document.cookie);
  var ca = decodedCookie.split(';');
  for(var i = 0; i <ca.length; i++) {
    var c = ca[i];
    while (c.charAt(0) == ' ') {
      c = c.substring(1);
    }
    if (c.indexOf(name) == 0) {
      return c.substring(name.length, c.length);
    }
  }
  return "";
}

function auto_fresh(t) {
  var outline = getCookie("outline");
  if (outline != "") {
    setCookie("outline", "manager", 30);
    console.log(outline);
  } else {
    location.replace("/login");
  }

  setInterval(function(){
    var bt = document.getElementById("button-refresh");
    if (bt.innerText == "REFRESH ON") {
      window.location.reload(1);
    } else {
    }
 }, t);
}
</script>

<script>
function set_refresh() {
  var bt = document.getElementById("button-refresh");
  if (bt.innerText == "REFRESH ON") {
    bt.innerText = "REFRESH OFF";
  } else {
    bt.innerText = "REFRESH ON";
  }
}
</script>

<script>
function alert_user(id) {
  alert(id);
}
</script>

<script>

function setCookie(cname, cvalue, exdays) {
  var d = new Date();
  d.setTime(d.getTime() + (exdays*24*60*60*1000));
  var expires = "expires="+ d.toUTCString();
  document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/;SameSite=Lax";
}

function exit() {
  setCookie("outline", "manager", -1);
  location.replace("/login");
}
</script>

</body>

</html>
`))
