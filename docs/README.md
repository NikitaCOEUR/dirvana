# Dirvana Documentation

This directory contains the Hugo-based documentation site for Dirvana.

## Local Development

### Prerequisites

You need **Hugo Extended** version 0.146.0 or later to build the site locally.

**Install Hugo Extended:**

- **macOS**: `brew install hugo`
- **Linux**: Download from [Hugo Releases](https://github.com/gohugoio/hugo/releases)
- **Windows**: `choco install hugo-extended`
- Or simply use hugo extended from current aqua installation.

Verify installation:
```bash
hugo version  # Should show "extended"
```

### Running Locally

```bash
cd docs
hugo server --buildDrafts
```

Then visit http://localhost:1313/dirvana/

### Building

```bash
cd docs
hugo --minify
```

The site will be generated in `docs/public/`.

## Deployment

The site is automatically deployed to GitHub Pages when changes are pushed to the `main` branch.

See `.github/workflows/deploy-docs.yml` for the deployment configuration.

## Theme

This site uses the [hugo-book](https://github.com/alex-shpak/hugo-book) theme.

## Structure

```
docs/
├── content/           # Markdown content
│   ├── _index.md     # Homepage
│   └── docs/         # Documentation pages
├── static/           # Static assets (images, etc.)
├── themes/           # Hugo themes (git submodule)
└── hugo.toml         # Site configuration
```

## Contributing

When adding new documentation:

1. Create a new `.md` file in `content/docs/`
2. Add front matter with `title` and `weight`
3. Write content in Markdown
4. Test locally with `hugo server`
5. Commit and push to trigger deployment

## Troubleshooting

### Error: Hugo not extended

If you see an error about SCSS/SASS, you need Hugo Extended:
```
Error: TOCSS: failed to transform "book.scss"
```

Solution: Ensure you're using Hugo Extended version.
