{
  description = "CLI fixture flake: devShell and prepare app";

  outputs = { self }: {
    devShells.x86_64-linux.default = { };
    apps.x86_64-linux.havn-session-prepare = {
      type = "app";
      program = "${self}/scripts/prepare-ok.sh";
    };
  };
}
