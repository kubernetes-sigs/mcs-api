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
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

type testWithRef struct {
	test string
	refs []string
}

var _ = ReportAfterSuite("MCS conformance report", func(report Report) {
	out, err := os.Create("report.txt")
	Expect(err).To(Succeed())

	passed := make(map[string][]testWithRef)
	failed := make(map[string][]testWithRef)

	for _, specReport := range report.SpecReports {
		var target *map[string][]testWithRef
		if specReport.State == types.SpecStatePassed {
			target = &passed
		} else {
			target = &failed
		}
		for _, label := range specReport.Labels() {
			existing := (*target)[label]
			reportEntries := specReport.ReportEntries
			refs := []string{}
			for _, reportEntry := range reportEntries {
				refs = append(refs, reportEntry.GetRawValue().(string))
			}

			(*target)[label] = append(existing, testWithRef{test: specReport.FullText(), refs: refs})
		}
	}

	for label, tests := range passed {
		fmt.Fprintf(out, "The implementation meets the following %s requirements:\n", label)
		for _, test := range tests {
			fmt.Fprintf(out, "⋅ %s (%v)\n", test.test, test.refs)
		}
	}

	for label, tests := range failed {
		fmt.Fprintf(out, "The implementation fails the following %s requirements:\n", label)
		for _, test := range tests {
			fmt.Fprintf(out, "⋅ %s (%v)\n", test.test, test.refs)
		}
	}
})
