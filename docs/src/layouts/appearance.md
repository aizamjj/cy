# Appearance

## Borders

Some nodes have a `:border` property that determines the appearance of the node's borders. The border property can be one of the following keywords:

- `:normal`
- `:rounded`
- `:block`
- `:outer-half`
- `:inner-half`
- `:thick`
- `:double`
- `:hidden`
- `:none`

`:hidden` still renders the border with blank cells; `:none` does not render a border at all.

## Text styling

{{story cast layout/styled}}

All string layout properties accept text styled with {{api style/render}} (or {{api style/text}}).

The layout shown in the monstrosity above was generated with the following code:

```janet
(def cmd1 (shell/new))
(def cmd2 (shell/new))
(layout/set
  (layout/new
    (margins
      (split
        (borders
          (attach :id cmd1)
          :border-fg "6"
          :title (style/text "some pane" :fg "0" :bg "6")
          :title-bottom (style/text "some subtitle" :fg "0" :bg "6"))
        (borders
          (pane :id cmd2)
          :border-fg "5"
          :title (style/text "some pane" :italic true :bg "5")
          :title-bottom (style/text "some subtitle" :italic true :bg "5"))
        :border-bg "3")
      :cols 70
      :border-bg "4")))
```
