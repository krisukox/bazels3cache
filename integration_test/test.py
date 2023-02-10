import subprocess, os, unittest


class TestBazelCache(unittest.TestCase):
    def setUp(self):
        s3_host = os.getenv("s3_host", "http://localhost:9444")
        self.bazels3cache = subprocess.Popen(
            [
                "/bazels3cache",
                "--s3url",
                "http://{0}/s3".format(s3_host),
                "--bucket",
                "bazel",
            ],
            stdout=subprocess.DEVNULL,
        )

    def test(self):
        bazel_test = [
            "bazel",
            "test",
            "//...",
            "--remote_cache=http://localhost:7777",
            "--remote_upload_local_results=true",
        ]
        bazel_clean = ["bazel", "clean"]
        test_workspace = "workspace"

        subprocess.run(bazel_test, cwd=test_workspace)
        subprocess.run(bazel_clean, cwd=test_workspace)
        result = subprocess.run(
            bazel_test,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            cwd=test_workspace,
        )
        print(result.stdout)
        self.assertEqual(result.stdout.find("12 remote cache hit"), -1)

    def tearDown(self):
        self.bazels3cache.kill()


if __name__ == "__main__":
    unittest.main()
