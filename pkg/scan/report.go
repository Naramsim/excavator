package scan

import (
	"html/template"
	"os"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type ReportInterface interface {
	Write(s *Scanner)
}

type HTMLReport struct {
	Outfile  string
	Template *template.Template
}
type YamlReport struct {
	Outfile string
}

func createFile(path string) *os.File {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal().
			Str("output", path).
			Err(err).
			Msg("Unable to create file")
	}
	return f
}

const (
	report = `
<html>
	<head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" /> 
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.1.3/css/bootstrap.min.css" integrity="sha384-MCw98/SFnGE8fJT3GXwEOngsV7Zt27NXFoaoApmYm81iuXoPkFOJwJ8ERdknLPMO" crossorigin="anonymous">
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/2.1.3/jquery.min.js"></script>
    <link href="https://gitcdn.github.io/bootstrap-toggle/2.2.2/css/bootstrap-toggle.min.css" rel="stylesheet">
    <script src="https://gitcdn.github.io/bootstrap-toggle/2.2.2/js/bootstrap-toggle.min.js"></script>
    <style>
.card {
  border: none;
  margin: 15px;
}

body, html {
  margin: 0;
  height:100%;
}
body{
  overflow: hidden;
}
.container-fluid, .parent{
  height: 100%;
}

#left, #right{
  position: relative;
  float: left;
  height:100%;
  overflow-y: auto; 
}

.blob-container {
  overflow-x: auto;
  overflow-y: hidden;
  padding: 15px;
}

.blob {
  border-spacing: 0;
  border-collapse: collapse;
  line-height: 0;
}

.blob-num {
  width: 1%;
  min-width: 50px;
  padding-right: 10px;
  padding-left: 10px;
  font-family: "SFMono-Regular,Consolas,Liberation Mono,Menlo,monospace";
  font-size: 12px;
  line-height: 20px;
  color: rgba(27,31,35,.3);
  text-align: right;
  white-space: nowrap;
}

.blob-code {
  white-space: pre;
  padding-left: 10px;
  padding-top: 0px;
}

a,a:hover {
  color: inherit;
  text-decoration: inherit;
  cursor: inherit;
}

.switch {
  position: relative;
  display: inline-block;
  width: 60px;
  height: 34px;
}

.switch input { 
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  -webkit-transition: .4s;
  transition: .4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 26px;
  width: 26px;
  left: 4px;
  bottom: 4px;
  background-color: white;
  -webkit-transition: .4s;
  transition: .4s;
}

input:checked + .slider {
  background-color: #2196F3;
}

input:focus + .slider {
  box-shadow: 0 0 1px #2196F3;
}

input:checked + .slider:before {
  -webkit-transform: translateX(26px);
  -ms-transform: translateX(26px);
  transform: translateX(26px);
}

.slider.round {
  border-radius: 34px;
}

.slider.round:before {
  border-radius: 50%;
}

.leaks {
  margin: 5%;
}
    </style>
  </head>

	<body>
  <div class="container-fluid">
    <div class="row parent">
      <div class="col-md-3" id="left">
        <div id="rules" class="text-center">
          <br/>
          <h3><span style="font-weight: bold; font-size: 40px;" id="rule-count">{{ len .RuleSet.Rules }}</span> active rule(s)</h3>
          <p>Defined by {{ .RulesPath }}<br/>last read on {{ .RuleSet.ReadAt | date "2006-01-02 15:04:05" }}</p>
          <div class="toggle-all d-flex justify-content-center">
            <button type="button" class="btn btn-primary" data-toggle="button" aria-pressed="false" autocomplete="off" onclick="enableAll()">
              Enable All
            </button>
            &nbsp;
            &nbsp;
            &nbsp;
            &nbsp;
            <button type="button" class="btn btn-primary" data-toggle="button" aria-pressed="false" autocomplete="off" onclick="disableAll()">
              Disable All
            </button>
          </div>
          <br/>
          <hr/>
          <br/>
          <br/>

          {{- with .RuleSet }}
          {{- range .Rules }}
            <div id="{{ sha1sum .Definition }}">
              <div class="card">
                <div class="card-body">
                  <h5 class="card-title">{{ .Category }}</h5>
                  <p class="card-text">{{ default "" .Description }}</p>
                  <samp>{{ .Definition }}</samp>
                  <br/>
                  <div class="container-fluid" style="text-align:center; padding-top: 15px;">
                    <label class="switch">
                      <input type="checkbox" name="checkbox" id="checkbox-{{ sha1sum .Definition}}" onclick="toggle({{ sha1sum .Definition }})" checked/>
                      <span class="slider"></span>
                    </label>
                  </div>
                </div>
                <hr/>
              </div>
            </div>
          {{- end }}
          {{- end }}
        </div>
      </div>
      <div class="col-md-9" id="right">
        <div class="leaks" id="leaks">
          <h1>Found <span style="font-weight: bold; font-size: 50px;" id="leak-count">{{ len .Result }}</span> potential credential leaks</h1>
          {{- $leaks := .Result }}
          {{- range .RuleSet.Rules }}
            <div id="container-{{ sha1sum .Definition }}">
              {{- $rule := . }}
              {{- range $leaks }}
              {{- if eq .Rule.Definition $rule.Definition }}
              <div class="card">
                <div class="card-body">
                  <h5 class="card-title">
                    {{ .File }}
                    <br>
                    {{ .Commit }}
                  </h5>
                  <p class="card-text">Author: {{ .Author }}    |   At: {{ .When | date "2006-01-02 15:04:05"}}</p>
                  <div class="blob-container table-responsive">
                    <table class="blob table-hover table-borderless">
                      <tbody>
                      {{- $start := . }}
                      {{- range $idx, $line := .Snippet }}
                      <tr>
                        <td class="blob-num">{{ add $idx $start.Line }}</td>
                        {{- if eq $idx $start.Affected }}
                        <td class="blob-code text-warning">
                        {{ $line }}
                        </td>
                        {{- else }}
                        <td class="blob-code">
                        {{ $line }}
                        </td>
                        {{- end }}
                      </tr>
                      {{- end }}
                      </tbody>
                    </table>
                  </div>
                </div>
                <hr/>
              </div>
              {{- end }}
              {{- end }}
            </div>
          {{- end }}
        </div>
      </div>
    </div>
  </div>
  <script>
    function toggle(checksum) {
      let check = document.getElementById("checkbox-"+checksum);
      let target = document.getElementById("container-"+checksum);
      let leakCount = document.getElementById("leak-count");
      let ruleCount = document.getElementById("rule-count");
      if (check.checked) {
        target.style.display = "block";
      } else {
        target.style.display = "none";
      }
      let leaks = document.getElementById("leaks").children;
      let activeLeaks = Array.from(leaks).filter(el => {
        if (el.tagName == "DIV") {
          return el.style.display != "none";
        }
        return false;
      });

      let totalLeak = 0;
      for (let leak of activeLeaks) {
        totalLeak += +leak.children.length;
      }
      ruleCount.innerHTML = activeLeaks.length;
      leakCount.innerHTML = totalLeak;
    };

    function enableAll() {
      let checkboxes = document.querySelectorAll('input[type=checkbox]');
      for (let checkbox of checkboxes) {
        checkbox.checked = false;
        checkbox.click();
      }
    }

    function disableAll() {
      let checkboxes = document.querySelectorAll('input[type=checkbox]');
      for (let checkbox of checkboxes) {
        checkbox.checked = true;
        checkbox.click();
      }
    }

  </script>
<script src="https://code.jquery.com/jquery-3.3.1.slim.min.js" integrity="sha384-q8i/X+965DzO0rT7abK41JStQIAqVgRVzpbzo5smXKp4YfRvH+8abtTE1Pi6jizo" crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.14.3/umd/popper.min.js" integrity="sha384-ZMP7rVo3mIykV+2+9J3UJ46jBk0WLaUAdn689aCwoqbBJiSnjAK/l8WvCWPIPm49" crossorigin="anonymous"></script>
<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.1.3/js/bootstrap.min.js" integrity="sha384-ChfqqxuZUCnJSK3+MXmPNIyE6ZbWh2IMqE241rYiqJxyMiZ6OW/JmZQ5stwEULTy" crossorigin="anonymous"></script>
	</body>
</html>	
`
)

func (h *HTMLReport) Write(s *Scanner) {
	h.Template = template.Must(template.New("report.gohtml").Funcs(
		sprig.FuncMap(),
	).Parse(report))
	h.Outfile = "index.html"
	f := createFile(h.Outfile)
	defer f.Close()
	if err := h.Template.Execute(f, s); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to execute template")
	}
	log.Info().
		Str("path", h.Outfile).
		Msg("Output has been written to")
}

func (y *YamlReport) Write(s *Scanner) {
	y.Outfile = time.Now().Format(time.RFC3339) + ".yaml"
	f := createFile(y.Outfile)
	defer f.Close()
	data, err := yaml.Marshal(&s)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Unable to marshal structure to yaml")
	}

	if _, err := f.Write(data); err != nil {
		log.Fatal().
			Str("output", y.Outfile).
			Err(err).
			Msg("Unable to write to file")
	}
	log.Info().
		Str("path", y.Outfile).
		Msg("Output has been written to")
}