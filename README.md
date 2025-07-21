# Postgresus Website

## CSS Development Workflow

This project uses TailwindCSS. **You must build CSS before committing.**

### Setup (one time)

```bash
npm install
```

### Development Process

1. **Make changes** to `index.html` or other HTML files
2. **Build CSS** before committing:
   ```bash
   npm run build-css
   ```
3. **Commit both files**:
   ```bash
   git add .
   git commit -m "Your changes"
   git push
   ```

### For Active Development

```bash
# Watch for changes (rebuilds automatically)
npm run watch-css
```

### Important Files

- `src/input.css` - Source CSS (edit this)
- `styles.min.css` - Generated CSS (don't edit manually)
- `index.html` - Uses the generated CSS

**Always run `npm run build-css` before pushing to GitHub!**
