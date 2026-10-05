# NieR-style icons

All icons use `fill="currentColor"`, so they take the text color of whatever they sit in.
Icons are 16×16. `pointer` is 20×12.

| file | use |
|---|---|
| pointer.svg | replaces the 👉 bullet / selected-row cursor |
| function.svg | functions |
| struct.svg | structs |
| constant.svg | constants |
| binding.svg | bindings / globals |
| entry-point.svg | @vertex / @fragment / @compute |
| import.svg | imports |
| group.svg | collapsible group header |

## Inline (recommended: colors follow the theme)

```html
<a class="toc-item" href="#fragment">
  <svg class="icon" viewBox="0 0 20 12" fill="currentColor" aria-hidden="true">
    <path fill-rule="evenodd" d="M0.5 6 L8 1.5 L15 6 L8 10.5 Z M8 4.4 A1.6 1.6 0 1 0 8 7.6 A1.6 1.6 0 1 0 8 4.4 Z"/>
    <rect x="17" y="1.8" width="2.4" height="2.4"/><rect x="17" y="7.8" width="2.4" height="2.4"/>
  </svg>
  fragment
</a>
```

```css
.icon { width: 1.25em; height: .75em; flex: none; vertical-align: middle; }
```

## As a CSS mask (keeps the markup free of SVG)

```css
.toc-item::before {
  content: "";
  display: inline-block;
  width: 20px; height: 12px;
  margin-right: 8px;
  background: currentColor;
  -webkit-mask: url(/icons/pointer.svg) no-repeat center / contain;
          mask: url(/icons/pointer.svg) no-repeat center / contain;
}
```
