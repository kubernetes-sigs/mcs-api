/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conformance

import (
	_ "embed"
	"html/template"
	"os"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

const (
	OptionalLabel      = "Optional"
	RequiredLabel      = "Required"
	SpecRefReportEntry = "spec-ref"
)

//go:embed report_template.gohtml
var reportHTML string

type testInfo struct {
	Desc string
	Ref  string
	Pass bool
}

type testGrouping struct {
	Name  string
	Tests []testInfo
}

var _ = ReportAfterSuite("MCS conformance report", func(report Report) {
	testGroupMap := map[string]*testGrouping{}

	for _, specReport := range report.SpecReports {
		if specReport.LeafNodeType != types.NodeTypeIt || specReport.State == types.SpecStatePending ||
			specReport.State == types.SpecStateSkipped {
			continue
		}

		for _, label := range specReport.Labels() {
			if testGroupMap[label] == nil {
				testGroupMap[label] = &testGrouping{
					Name: label,
				}
			}

			ref := ""
			for i := range specReport.ReportEntries {
				if specReport.ReportEntries[i].Name == SpecRefReportEntry {
					ref = specReport.ReportEntries[i].GetRawValue().(string)
					break
				}
			}

			testGroupMap[label].Tests = append(testGroupMap[label].Tests, testInfo{
				Desc: specReport.FullText(),
				Ref:  ref,
				Pass: specReport.State == types.SpecStatePassed,
			})
		}
	}

	var testGroups []testGrouping

	for _, g := range testGroupMap {
		testGroups = append(testGroups, *g)
	}

	data := struct {
		Groups []testGrouping
	}{
		testGroups,
	}

	out, err := os.Create("report.html")
	Expect(err).To(Succeed())

	tmpl, err := template.New("report").Parse(reportHTML)
	Expect(err).To(Succeed())

	err = tmpl.Execute(out, data)
	Expect(err).To(Succeed())
})
