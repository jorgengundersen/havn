{
  description = "CLI fixture flake: missing required devShell";

  outputs = { self }: {
    apps.x86_64-linux.havn-session-prepare = {
      type = "app";
      program = "${self}/scripts/prepare-ok.sh";
    };
  };
}
