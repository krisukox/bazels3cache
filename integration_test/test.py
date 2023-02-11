import subprocess, os, unittest


class TestBazelCache(unittest.TestCase):
    def test(self):
        s3_host = os.getenv("s3_host", "localhost:9444")
        self.bazels3cache = subprocess.run(
            [
                "/bazels3cache",
                "--s3url",
                "http://{0}/s3".format(s3_host),
                "--bucket",
                "bazel",
            ],
            check=True,
        )

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
        self.assertNotEqual(result.stdout.find("12 remote cache hit"), -1)

        self.bazels3cache = subprocess.run(["/bazels3cache", "--stop"], check=True)


if __name__ == "__main__":
    unittest.main()
