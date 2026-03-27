class Schmournal < Formula
  desc "Terminal-based work journal"
  homepage "https://github.com/SleepyPxnda/schmournal"
  url "https://github.com/SleepyPxnda/schmournal/archive/refs/tags/v1.9.2.tar.gz"
  sha256 "a40097fc51479dd5d20f0123d08045804202169a65e93ecce8ab597fa6ab7e0a"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  def post_install
    config_dir = "#{Dir.home}/.config"
    config_file = "#{config_dir}/schmournal.config"
    return if File.exist?(config_file)

    FileUtils.mkdir_p(config_dir)
    File.write(config_file, <<~TOML)
      # Schmournal Configuration
      # Location: ~/.config/schmournal.config

      # Directory where journal JSON files are stored.
      # The ~ is expanded to your home directory.
      storage_path = "~/.journal"

      # ── Keybinds ──────────────────────────────────────────────────────────────────
      # Each value is a single key string as understood by the terminal
      # (e.g. "q", "x", "ctrl+s").  Arrow keys, Enter, Esc and Tab are not
      # configurable here — they always keep their default role.

      [keybinds.list]
      quit       = "q"   # Quit the application
      open_today = "n"   # Open / create today's entry
      open_date  = "c"   # Open / create an entry for a specific date
      delete     = "d"   # Delete the selected day record
      export     = "x"   # Export the selected day to Markdown
      stats_view = "s"   # Open the stats overview

      [keybinds.day]
      add_work         = "w"   # Add a new work entry
      add_break        = "b"   # Add a new break entry
      edit             = "e"   # Edit selected entry (or open notes when none selected)
      delete           = "d"   # Delete selected entry (or the whole day when none selected)
      set_start_now    = "s"   # Set start time to now
      set_start_manual = "S"   # Set start time manually
      set_end_now      = "f"   # Set end time to now
      set_end_manual   = "F"   # Set end time manually
      notes            = "n"   # Open the notes editor
      todo_overview    = "t"   # Toggle TODO pane in day view
      export           = "x"   # Export day to Markdown
    TOML
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/schmournal --version")
  end
end
