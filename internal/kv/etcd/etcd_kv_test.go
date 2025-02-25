// Copyright (C) 2019-2020 Zilliz. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under the License.

package etcdkv_test

import (
	"os"
	"strings"
	"testing"
	"time"

	etcdkv "github.com/milvus-io/milvus/internal/kv/etcd"
	"github.com/milvus-io/milvus/internal/util/paramtable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var Params paramtable.BaseTable

func TestMain(m *testing.M) {
	Params.Init()
	code := m.Run()
	os.Exit(code)
}

func TestEtcdKV_Load(te *testing.T) {
	endpoints, err := Params.Load("_EtcdEndpoints")
	if err != nil {
		panic(err)
	}

	etcdEndPoints := strings.Split(endpoints, ",")

	te.Run("EtcdKV SaveAndLoad", func(t *testing.T) {
		rootPath := "/etcd/test/root/saveandload"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		require.NoError(t, err)
		err = etcdKV.RemoveWithPrefix("")
		require.NoError(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		saveAndLoadTests := []struct {
			key   string
			value string
		}{
			{"test1", "value1"},
			{"test2", "value2"},
			{"test1/a", "value_a"},
			{"test1/b", "value_b"},
		}

		for i, test := range saveAndLoadTests {
			if i < 4 {
				err = etcdKV.Save(test.key, test.value)
				assert.NoError(t, err)
			}

			val, err := etcdKV.Load(test.key)
			assert.NoError(t, err)
			assert.Equal(t, test.value, val)
		}

		invalidLoadTests := []struct {
			invalidKey string
		}{
			{"t"},
			{"a"},
			{"test1a"},
		}

		for _, test := range invalidLoadTests {
			val, err := etcdKV.Load(test.invalidKey)
			assert.Error(t, err)
			assert.Zero(t, val)
		}

		loadPrefixTests := []struct {
			prefix string

			expectedKeys   []string
			expectedValues []string
			expectedError  error
		}{
			{"test", []string{
				etcdKV.GetPath("test1"),
				etcdKV.GetPath("test2"),
				etcdKV.GetPath("test1/a"),
				etcdKV.GetPath("test1/b")}, []string{"value1", "value2", "value_a", "value_b"}, nil},
			{"test1", []string{
				etcdKV.GetPath("test1"),
				etcdKV.GetPath("test1/a"),
				etcdKV.GetPath("test1/b")}, []string{"value1", "value_a", "value_b"}, nil},
			{"test2", []string{etcdKV.GetPath("test2")}, []string{"value2"}, nil},
			{"", []string{
				etcdKV.GetPath("test1"),
				etcdKV.GetPath("test2"),
				etcdKV.GetPath("test1/a"),
				etcdKV.GetPath("test1/b")}, []string{"value1", "value2", "value_a", "value_b"}, nil},
			{"test1/a", []string{etcdKV.GetPath("test1/a")}, []string{"value_a"}, nil},
			{"a", []string{}, []string{}, nil},
			{"root", []string{}, []string{}, nil},
			{"/etcd/test/root", []string{}, []string{}, nil},
		}

		for _, test := range loadPrefixTests {
			actualKeys, actualValues, err := etcdKV.LoadWithPrefix(test.prefix)
			assert.ElementsMatch(t, test.expectedKeys, actualKeys)
			assert.ElementsMatch(t, test.expectedValues, actualValues)
			assert.Equal(t, test.expectedError, err)

			actualKeys, actualValues, versions, err := etcdKV.LoadWithPrefix2(test.prefix)
			assert.ElementsMatch(t, test.expectedKeys, actualKeys)
			assert.ElementsMatch(t, test.expectedValues, actualValues)
			assert.NotZero(t, versions)
			assert.Equal(t, test.expectedError, err)
		}

		removeTests := []struct {
			validKey   string
			invalidKey string
		}{
			{"test1", "abc"},
			{"test1/a", "test1/lskfjal"},
			{"test1/b", "test1/b"},
			{"test2", "-"},
		}

		for _, test := range removeTests {
			err = etcdKV.Remove(test.validKey)
			assert.NoError(t, err)

			_, err = etcdKV.Load(test.validKey)
			assert.Error(t, err)

			err = etcdKV.Remove(test.validKey)
			assert.NoError(t, err)
			err = etcdKV.Remove(test.invalidKey)
			assert.NoError(t, err)
		}
	})

	te.Run("EtcdKV LoadWithRevision", func(t *testing.T) {
		rootPath := "/etcd/test/root/LoadWithRevision"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		assert.Nil(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		prepareKV := []struct {
			inKey   string
			inValue string
		}{
			{"a", "a_version1"},
			{"b", "b_version2"},
			{"a", "a_version3"},
			{"c", "c_version4"},
			{"a/suba", "a_version5"},
		}

		for _, test := range prepareKV {
			err = etcdKV.Save(test.inKey, test.inValue)
			require.NoError(t, err)
		}

		loadWithRevisionTests := []struct {
			inKey string

			expectedKeyNo  int
			expectedValues []string
		}{
			{"a", 2, []string{"a_version3", "a_version5"}},
			{"b", 1, []string{"b_version2"}},
			{"c", 1, []string{"c_version4"}},
		}

		for _, test := range loadWithRevisionTests {
			keys, values, revision, err := etcdKV.LoadWithRevision(test.inKey)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedKeyNo, len(keys))
			assert.ElementsMatch(t, test.expectedValues, values)
			assert.NotZero(t, revision)
		}

	})

	te.Run("EtcdKV MultiSaveAndMultiLoad", func(t *testing.T) {
		rootPath := "/etcd/test/root/multi_save_and_multi_load"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		assert.Nil(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		multiSaveTests := map[string]string{
			"key_1":      "value_1",
			"key_2":      "value_2",
			"key_3/a":    "value_3a",
			"multikey_1": "multivalue_1",
			"multikey_2": "multivalue_2",
			"_":          "other",
		}

		err = etcdKV.MultiSave(multiSaveTests)
		assert.NoError(t, err)
		for k, v := range multiSaveTests {
			actualV, err := etcdKV.Load(k)
			assert.NoError(t, err)
			assert.Equal(t, v, actualV)
		}

		multiLoadTests := []struct {
			inputKeys      []string
			expectedValues []string
		}{
			{[]string{"key_1"}, []string{"value_1"}},
			{[]string{"key_1", "key_2", "key_3/a"}, []string{"value_1", "value_2", "value_3a"}},
			{[]string{"multikey_1", "multikey_2"}, []string{"multivalue_1", "multivalue_2"}},
			{[]string{"_"}, []string{"other"}},
		}

		for _, test := range multiLoadTests {
			vs, err := etcdKV.MultiLoad(test.inputKeys)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedValues, vs)
		}

		invalidMultiLoad := []struct {
			invalidKeys    []string
			expectedValues []string
		}{
			{[]string{"a", "key_1"}, []string{"", "value_1"}},
			{[]string{".....", "key_1"}, []string{"", "value_1"}},
			{[]string{"*********"}, []string{""}},
			{[]string{"key_1", "1"}, []string{"value_1", ""}},
		}

		for _, test := range invalidMultiLoad {
			vs, err := etcdKV.MultiLoad(test.invalidKeys)
			assert.Error(t, err)
			assert.Equal(t, test.expectedValues, vs)
		}

		removeWithPrefixTests := []string{
			"key_1",
			"multi",
		}

		for _, k := range removeWithPrefixTests {
			err = etcdKV.RemoveWithPrefix(k)
			assert.NoError(t, err)

			ks, vs, err := etcdKV.LoadWithPrefix(k)
			assert.Empty(t, ks)
			assert.Empty(t, vs)
			assert.NoError(t, err)
		}

		multiRemoveTests := []string{
			"key_2",
			"key_3/a",
			"multikey_2",
			"_",
		}

		err = etcdKV.MultiRemove(multiRemoveTests)
		assert.NoError(t, err)

		ks, vs, err := etcdKV.LoadWithPrefix("")
		assert.NoError(t, err)
		assert.Empty(t, ks)
		assert.Empty(t, vs)

		multiSaveAndRemoveTests := []struct {
			multiSaves   map[string]string
			multiRemoves []string
		}{
			{map[string]string{"key_1": "value_1"}, []string{}},
			{map[string]string{"key_2": "value_2"}, []string{"key_1"}},
			{map[string]string{"key_3/a": "value_3a"}, []string{"key_2"}},
			{map[string]string{"multikey_1": "multivalue_1"}, []string{}},
			{map[string]string{"multikey_2": "multivalue_2"}, []string{"multikey_1", "key_3/a"}},
			{make(map[string]string), []string{"multikey_2"}},
		}
		for _, test := range multiSaveAndRemoveTests {
			err = etcdKV.MultiSaveAndRemove(test.multiSaves, test.multiRemoves)
			assert.NoError(t, err)
		}

		ks, vs, err = etcdKV.LoadWithPrefix("")
		assert.NoError(t, err)
		assert.Empty(t, ks)
		assert.Empty(t, vs)
	})

	te.Run("EtcdKV MultiRemoveWithPrefix", func(t *testing.T) {
		rootPath := "/etcd/test/root/multi_remove_with_prefix"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		require.NoError(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		prepareTests := map[string]string{
			"x/abc/1": "1",
			"x/abc/2": "2",
			"x/def/1": "10",
			"x/def/2": "20",
			"x/den/1": "100",
			"x/den/2": "200",
		}

		err = etcdKV.MultiSave(prepareTests)
		require.NoError(t, err)

		multiRemoveWithPrefixTests := []struct {
			prefix []string

			testKey       string
			expectedValue string
		}{
			{[]string{"x/abc"}, "x/abc/1", ""},
			{[]string{}, "x/abc/2", ""},
			{[]string{}, "x/def/1", "10"},
			{[]string{}, "x/def/2", "20"},
			{[]string{}, "x/den/1", "100"},
			{[]string{}, "x/den/2", "200"},
			{[]string{}, "not-exist", ""},
			{[]string{"x/def", "x/den"}, "x/def/1", ""},
			{[]string{}, "x/def/1", ""},
			{[]string{}, "x/def/2", ""},
			{[]string{}, "x/den/1", ""},
			{[]string{}, "x/den/2", ""},
			{[]string{}, "not-exist", ""},
		}

		for _, test := range multiRemoveWithPrefixTests {
			if len(test.prefix) > 0 {
				err = etcdKV.MultiRemoveWithPrefix(test.prefix)
				assert.NoError(t, err)
			}

			v, _ := etcdKV.Load(test.testKey)
			assert.Equal(t, test.expectedValue, v)
		}

		k, v, err := etcdKV.LoadWithPrefix("/")
		assert.NoError(t, err)
		assert.Zero(t, len(k))
		assert.Zero(t, len(v))

		// MultiSaveAndRemoveWithPrefix
		err = etcdKV.MultiSave(prepareTests)
		require.NoError(t, err)
		multiSaveAndRemoveWithPrefixTests := []struct {
			multiSave map[string]string
			prefix    []string

			loadPrefix         string
			lengthBeforeRemove int
			lengthAfterRemove  int
		}{
			{map[string]string{}, []string{"x/abc", "x/def", "x/den"}, "x", 6, 0},
			{map[string]string{"y/a": "vvv", "y/b": "vvv"}, []string{}, "y", 0, 2},
			{map[string]string{"y/c": "vvv"}, []string{}, "y", 2, 3},
			{map[string]string{"p/a": "vvv"}, []string{"y/a", "y"}, "y", 3, 0},
			{map[string]string{}, []string{"p"}, "p", 1, 0},
		}

		for _, test := range multiSaveAndRemoveWithPrefixTests {
			k, _, err = etcdKV.LoadWithPrefix(test.loadPrefix)
			assert.NoError(t, err)
			assert.Equal(t, test.lengthBeforeRemove, len(k))

			err = etcdKV.MultiSaveAndRemoveWithPrefix(test.multiSave, test.prefix)
			assert.NoError(t, err)

			k, _, err = etcdKV.LoadWithPrefix(test.loadPrefix)
			assert.NoError(t, err)
			assert.Equal(t, test.lengthAfterRemove, len(k))
		}
	})

	te.Run("EtcdKV Watch", func(t *testing.T) {
		rootPath := "/etcd/test/root/watch"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		assert.Nil(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		ch := etcdKV.Watch("x")
		resp := <-ch
		assert.True(t, resp.Created)

		ch = etcdKV.WatchWithPrefix("x")
		resp = <-ch
		assert.True(t, resp.Created)
	})

	te.Run("Etcd Revision", func(t *testing.T) {
		rootPath := "/etcd/test/root/watch"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		assert.Nil(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		revisionTests := []struct {
			inKey       string
			fistValue   string
			secondValue string
		}{
			{"a", "v1", "v11"},
			{"y", "v2", "v22"},
			{"z", "v3", "v33"},
		}

		for _, test := range revisionTests {
			err = etcdKV.Save(test.inKey, test.fistValue)
			require.NoError(t, err)

			_, _, revision, _ := etcdKV.LoadWithRevision(test.inKey)
			ch := etcdKV.WatchWithRevision(test.inKey, revision+1)

			err = etcdKV.Save(test.inKey, test.secondValue)
			require.NoError(t, err)

			resp := <-ch
			assert.Equal(t, 1, len(resp.Events))
			assert.Equal(t, test.secondValue, string(resp.Events[0].Kv.Value))
			assert.Equal(t, revision+1, resp.Header.Revision)
		}

		err = etcdKV.CompareVersionAndSwap("a/b/c", 0, "1")
		assert.NoError(t, err)

		value, err := etcdKV.Load("a/b/c")
		assert.NoError(t, err)
		assert.Equal(t, value, "1")

		err = etcdKV.CompareVersionAndSwap("a/b/c", 0, "1")
		assert.Error(t, err)

		err = etcdKV.CompareValueAndSwap("a/b/c", "1", "2")
		assert.NoError(t, err)

		err = etcdKV.CompareValueAndSwap("a/b/c", "1", "2")
		assert.Error(t, err)
	})

	te.Run("Etcd Lease", func(t *testing.T) {
		rootPath := "/etcd/test/root/lease"
		etcdKV, err := etcdkv.NewEtcdKV(etcdEndPoints, rootPath)
		assert.Nil(t, err)

		defer etcdKV.Close()
		defer etcdKV.RemoveWithPrefix("")

		leaseID, err := etcdKV.Grant(10)
		assert.NoError(t, err)

		etcdKV.KeepAlive(leaseID)

		tests := map[string]string{
			"a/b":   "v1",
			"a/b/c": "v2",
			"x":     "v3",
		}

		for k, v := range tests {
			err = etcdKV.SaveWithLease(k, v, leaseID)
			assert.NoError(t, err)

			err = etcdKV.SaveWithLease(k, v, clientv3.LeaseID(999))
			assert.Error(t, err)
		}

	})
}

func TestElapse(t *testing.T) {
	start := time.Now()
	isElapse := etcdkv.CheckElapseAndWarn(start, "err message")
	assert.Equal(t, isElapse, false)

	time.Sleep(2001 * time.Millisecond)
	isElapse = etcdkv.CheckElapseAndWarn(start, "err message")
	assert.Equal(t, isElapse, true)
}
