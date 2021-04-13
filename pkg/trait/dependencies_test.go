/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trait

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
)

func TestDependenciesTraitApplicability(t *testing.T) {
	e := &Environment{
		Catalog:     NewEnvironmentTestCatalog(),
		Integration: &v1.Integration{},
	}

	trait := newDependenciesTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)

	e.Integration.Status.Phase = v1.IntegrationPhaseNone
	enabled, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)

	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	enabled, err = trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)
}

func TestIntegrationDefaultDeps(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "Request.java",
							Content: `from("direct:foo").to("log:bar");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newDependenciesTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-java-joor-dsl",
			"mvn:org.apache.camel.k:camel-k-runtime"},
		e.Integration.Status.Dependencies,
	)
}

func TestIntegrationCustomDeps(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Dependencies: []string{
					"camel:netty-http",
					"org.foo:bar",
				},
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "Request.java",
							Content: `from("direct:foo").to("log:bar");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newDependenciesTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.ElementsMatch(t,
		[]string{
			"camel:direct",
			"camel:log",
			"camel:netty-http",
			"org.foo:bar",
			"mvn:org.apache.camel.quarkus:camel-quarkus-java-joor-dsl",
			"mvn:org.apache.camel.k:camel-k-runtime"},
		e.Integration.Status.Dependencies,
	)
}

func TestIntegrationAutoGeneratedDeps(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "Request.java",
							Content: `from("direct:foo").to("log:bar");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
				GeneratedSources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "RequestAuto.xml",
							Content: `<rests xmlns="http://camel.apache.org/schema/spring"><rest path="/camel/"></rest></rests>`,
						},
						Language: v1.LanguageXML,
					},
				},
			},
		},
	}

	for _, trait := range []Trait{newInitTrait(), newDependenciesTrait()} {
		enabled, err := trait.Configure(e)
		assert.Nil(t, err)
		assert.True(t, enabled)
		assert.Nil(t, trait.Apply(e))
	}

	for _, processor := range e.PostStepProcessors {
		assert.Nil(t, processor(e))
	}

	assert.ElementsMatch(
		t,
		[]string{
			"camel:direct",
			"camel:log",
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-java-joor-dsl",
			"mvn:org.apache.camel.quarkus:camel-quarkus-xml-io-dsl",
			"mvn:org.apache.camel.k:camel-k-runtime",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http"},
		e.Integration.Status.Dependencies,
	)
}

// in this test the language of the source is something unrelated to the
// loader to test the order in which loader and language are taken into
// account when dependencies are computed
func TestIntegrationCustomLoader(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "flow.java",
							Content: `from("direct:foo").to("log:bar");`,
						},
						Language: v1.LanguageJavaSource,
						Loader:   "yaml",
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newDependenciesTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.ElementsMatch(t,
		[]string{
			"camel:direct",
			"camel:log",
			"mvn:org.apache.camel.k:camel-k-loader-yaml",
			"mvn:org.apache.camel.k:camel-k-runtime"},
		e.Integration.Status.Dependencies,
	)
}

func TestRestDeps(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "flow.java",
							Content: `rest().to("log:bar");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newDependenciesTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.Subset(
		t,
		e.Integration.Status.Dependencies,
		[]string{
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
	)
}

func TestRestDepsQuarkus(t *testing.T) {
	catalog, err := camel.QuarkusCatalog()
	assert.Nil(t, err)

	e := &Environment{
		Catalog:      NewEnvironmentTestCatalog(),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "flow.java",
							Content: `rest().route().to("log:bar");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	}

	trait := newDependenciesTrait()
	enabled, err := trait.Configure(e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(e)
	assert.Nil(t, err)
	assert.Subset(
		t,
		e.Integration.Status.Dependencies,
		[]string{
			"mvn:org.apache.camel.quarkus:camel-quarkus-rest",
			"mvn:org.apache.camel.quarkus:camel-quarkus-platform-http",
		},
	)
}
