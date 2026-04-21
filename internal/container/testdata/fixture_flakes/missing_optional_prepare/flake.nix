{
  description = "Fixture flake: required devShell only";

  outputs = { }: {
    devShells.x86_64-linux.default = { };
  };
}
