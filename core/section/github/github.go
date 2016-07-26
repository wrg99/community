// Copyright 2016 Documize Inc. <legal@documize.com>. All rights reserved.
//
// This software (Documize Community Edition) is licensed under
// GNU AGPL v3 http://www.gnu.org/licenses/agpl-3.0.en.html
//
// You can operate outside the AGPL restrictions by purchasing
// Documize Enterprise Edition and obtaining a commercial license
// by contacting <sales@documize.com>.
//
// https://documize.com

package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/documize/community/core/log"
	"github.com/documize/community/core/section/provider"
)

// TODO find a smaller image than the one below
const githubGravatar = "https://i2.wp.com/assets-cdn.github.com/images/gravatars/gravatar-user-420.png"

var meta provider.TypeMeta

func init() {
	meta = provider.TypeMeta{}

	meta.ID = "38c0e4c5-291c-415e-8a4d-262ee80ba5df"
	meta.Title = "GitHub"
	meta.Description = "Link code commits and issues"
	meta.ContentType = "github"
	meta.Callback = Callback
}

// Provider represents GitHub
type Provider struct {
}

// Meta describes us.
func (*Provider) Meta() provider.TypeMeta {
	return meta
}

// Command to run the various functions required...
func (p *Provider) Command(ctx *provider.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	method := query.Get("method")

	if len(method) == 0 {
		msg := "missing method name"
		log.ErrorString("github: " + msg)
		provider.WriteMessage(w, "gitub", msg)
		return
	}

	if method == "config" {
		var ret struct {
			CID string `json:"clientID"`
			URL string `json:"authorizationCallbackURL"`
		}
		ret.CID = clientID()
		ret.URL = authorizationCallbackURL()
		provider.WriteJSON(w, ret)
		return
	}

	defer r.Body.Close() // ignore error

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "Bad body"
		log.ErrorString("github: " + msg)
		provider.WriteMessage(w, "github", msg)
		return
	}

	if method == "saveSecret" { // secret Token update code

		// write the new one, direct from JS
		if err = ctx.SaveSecrets(string(body)); err != nil {
			log.Error("github settoken configuration", err)
			provider.WriteError(w, "github", err)
			return
		}
		provider.WriteEmpty(w)
		return

	}

	// load the config from the client-side
	config := githubConfig{}
	err = json.Unmarshal(body, &config)

	if err != nil {
		log.Error("github Command Unmarshal", err)
		provider.WriteError(w, "github", err)
		return
	}

	config.Clean()
	// always use DB version of the token
	config.Token = ctx.GetSecrets("token") // get the secret token in the database

	client := p.githubClient(config)

	switch method {

	case "checkAuth":

		if len(config.Token) == 0 {
			err = errors.New("empty github token")
		} else {
			err = validateToken(config.Token)
		}
		if err != nil {
			// token now invalid, so wipe it
			ctx.SaveSecrets("") // ignore error, already in an error state
			log.Error("github check token validation", err)
			provider.WriteError(w, "github", err)
			return
		}
		provider.WriteEmpty(w)

	default:

		if listFailed(method, config, client, w) {

			rep, ok := reports[method]
			if !ok {
				log.ErrorString("Github connector unknown method: " + method)
				provider.WriteEmpty(w)
			}
			rep.command(p, client, config, w)

		}

	}
}

// Refresh ... gets the latest version
func (p *Provider) Refresh(ctx *provider.Context, configJSON, data string) string {
	var c = githubConfig{}

	err := json.Unmarshal([]byte(configJSON), &c)

	if err != nil {
		log.Error("unable to unmarshall github config", err)
		return "internal configuration error '" + err.Error() + "'"
	}

	c.Clean()
	c.Token = ctx.GetSecrets("token")

	rep, ok := reports[c.ReportInfo.ID]
	if !ok {
		msg := "github report not found for: " + c.ReportInfo.ID
		log.ErrorString(msg)
		return "Documize internal error: " + msg
	}

	return rep.refresh(p, c, data)

}

// Render ... just returns the data given, suitably formatted
func (p *Provider) Render(ctx *provider.Context, config, data string) string {
	var err error

	payload := githubRender{}
	var c = githubConfig{}

	err = json.Unmarshal([]byte(config), &c)

	if err != nil {
		log.Error("unable to unmarshall github config", err)
		return "Please delete and recreate this Github section."
	}

	c.Clean()
	c.Token = ctx.GetSecrets("token")

	payload.Config = c
	payload.Repo = c.RepoInfo
	payload.Limit = c.BranchLines
	if len(c.BranchSince) > 0 {
		payload.DateMessage = "created after " + c.BranchSince
	}

	rep, ok := reports[c.ReportInfo.ID]
	if !ok {
		msg := "github report not found for: " + c.ReportInfo.ID
		log.ErrorString(msg)
		return "Documize internal error: " + msg
	}

	if err = rep.render(&c, &payload, data); err != nil {
		log.Error("unable to render "+c.ReportInfo.ID, err)
		return "Documize internal github render " + c.ReportInfo.ID + " error: " + err.Error() + "<BR>" + data
	}

	t := template.New("github")

	t, err = t.Parse(rep.template)

	if err != nil {
		log.Error("github render template.Parse error:", err)
		return "Documize internal github template.Parse error: " + err.Error()
	}

	buffer := new(bytes.Buffer)
	err = t.Execute(buffer, payload)
	if err != nil {
		log.Error("github render template.Execute error:", err)
		return "Documize internal github template.Execute error: " + err.Error()
	}

	return buffer.String()
}