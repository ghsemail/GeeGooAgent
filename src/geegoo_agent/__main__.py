"""Allow ``python -m geegoo`` before ``geegoo`` entry point is installed."""

from geegoo_agent.cli import main

if __name__ == "__main__":
    raise SystemExit(main())
