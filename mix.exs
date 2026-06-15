defmodule Xrc.MixProject do
  use Mix.Project

  def project do
    [
      app: :xrc,
      version: "0.1.0",
      elixir: "~> 1.19",
      start_permanent: Mix.env() == :prod,
      escript: escript(),
      deps: deps()
    ]
  end

  def application do
    [
      extra_applications: [:logger, :inets, :ssl]
    ]
  end

  defp escript do
    [main_module: Xrc.CLI, name: "xrc"]
  end

  defp deps do
    [
      {:req, "~> 0.5"},
      {:jason, "~> 1.4"},
      {:owl, "~> 0.12"}
    ]
  end
end
