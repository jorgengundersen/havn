{
  description = "Fixture flake: devShell and failing prepare app";

  outputs = { self }: {
    devShells.x86_64-linux.default = { };
    apps.x86_64-linux.havn-session-prepare = {
      type = "app";
      program = "${self}/scripts/prepare-fail.sh";
    };
  };
}
