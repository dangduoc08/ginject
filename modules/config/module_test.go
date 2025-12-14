package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	envFilename := ".env.test"
	envFilePath, _ := findRootDir(envFilename)

	output1 := loadDotENV(filepath.Join(envFilePath, envFilename), true)
	expect1 := map[string]string{}
	expect1["KEY_1"] = "1"
	expect1["KEY2"] = "22"
	expect1["key_3"] = "333"
	expect1["key4"] = "4444"
	expect1["KEY_5"] = "666666"
	expect1["KEY_6"] = "#999=99 10"
	expect1["KEY_8"] = "88888888"
	expect1["KEY_9"] = expect1["KEY_8"]
	expect1["KEY_10"] = expect1["KEY_6"]
	expect1["KEY_11"] = expect1["KEY_1"] + "1234567_" + expect1["KEY2"]
	expect1["KEY_12"] = expect1["KEY_11"] + "_abc_xyz_" + expect1["KEY_10"]
	expect1["PRIVATE_KEY_1"] = "-----BEGIN RSA PRIVATE KEY-----\n...\n" + expect1["KEY_6"] + "\n...\n-----END DSA PRIVATE KEY-----\n"
	expect1["PRIVATE_KEY_2"] = "-----BEGIN RSA PRIVATE KEY-----\n" +
		"...\n" +
		expect1["KEY_1"] + "_" + expect1["KEY_5"] + "\n" +
		"...\n" +
		"-----END DSA PRIVATE KEY-----"

	if len(output1) != len(expect1) {
		t.Errorf("loadDotENV len(output1) output  = %v; len(expect1) expected = %v", len(output1), len(expect1))
	}

	if output1["KEY_1"] != expect1["KEY_1"] {
		t.Errorf("loadDotENV output1[\"KEY_1\"] output  = %v; expect1[\"KEY_1\"] expected = %v", output1["KEY_1"], expect1["KEY_1"])
	}

	if output1["KEY2"] != expect1["KEY2"] {
		t.Errorf("loadDotENV output1[\"KEY2\"] output  = %v; expect1[\"KEY2\"] expected = %v", output1["KEY2"], expect1["KEY2"])
	}

	if output1["key_3"] != expect1["key_3"] {
		t.Errorf("loadDotENV output1[\"key_3\"] output  = %v; expect1[\"key_3\"] expected = %v", output1["key_3"], expect1["key_3"])
	}

	if output1["key4"] != expect1["key4"] {
		t.Errorf("loadDotENV output1[\"key4\"] output  = %v; expect1[\"key4\"] expected = %v", output1["key4"], expect1["key4"])
	}

	if output1["KEY_5"] != expect1["KEY_5"] {
		t.Errorf("loadDotENV output1[\"KEY_5\"] output  = %v; expect1[\"KEY_5\"] expected = %v", output1["KEY_5"], expect1["KEY_5"])
	}

	if output1["KEY_6"] != expect1["KEY_6"] {
		t.Errorf("loadDotENV output1[\"KEY_6\"] output  = %v; expect1[\"KEY_6\"] expected = %v", output1["KEY_6"], expect1["KEY_6"])
	}

	if output1["KEY_8"] != expect1["KEY_8"] {
		t.Errorf("loadDotENV output1[\"KEY_8\"] output  = %v; expect1[\"KEY_8\"] expected = %v", output1["KEY_8"], expect1["KEY_8"])
	}

	if output1["KEY_9"] != expect1["KEY_9"] {
		t.Errorf("loadDotENV output1[\"KEY_9\"] output  = %v; expect1[\"KEY_9\"] expected = %v", output1["KEY_9"], expect1["KEY_9"])
	}

	if output1["KEY_10"] != expect1["KEY_10"] {
		t.Errorf("loadDotENV output1[\"KEY_10\"] output  = %v; expect1[\"KEY_10\"] expected = %v", output1["KEY_10"], expect1["KEY_10"])
	}

	if output1["KEY_11"] != expect1["KEY_11"] {
		t.Errorf("loadDotENV output1[\"KEY_11\"] output  = %v; expect1[\"KEY_11\"] expected = %v", output1["KEY_11"], expect1["KEY_11"])
	}

	if output1["KEY_12"] != expect1["KEY_12"] {
		t.Errorf("loadDotENV output1[\"KEY_12\"] output  = %v; expect1[\"KEY_12\"] expected = %v", output1["KEY_12"], expect1["KEY_12"])
	}

	if output1["PRIVATE_KEY_1"] != expect1["PRIVATE_KEY_1"] {
		t.Errorf("loadDotENV output1[\"PRIVATE_KEY_1\"] output  = %v; expect1[\"PRIVATE_KEY_1\"] expected = %v", output1["PRIVATE_KEY_1"], expect1["PRIVATE_KEY_1"])
	}

	if output1["PRIVATE_KEY_2"] != expect1["PRIVATE_KEY_2"] {
		t.Errorf("loadDotENV output1[\"PRIVATE_KEY_2\"] output  = %v; expect1[\"PRIVATE_KEY_2\"] expected = %v", output1["PRIVATE_KEY_2"], expect1["PRIVATE_KEY_2"])
	}
}
