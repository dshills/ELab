# ELab artwork

`app-icon.svg` is original ELab artwork: an E-shaped circuit, resistor and amber
LED on a blue instrument tile. It stays recognizable without text and has a
transparent margin for macOS icon placement. The same artwork is used in every
ELab theme; this is application identity, not a toolbar symbol.

`app-icon.png` is the checked-in 1024×1024 RGBA export. The Vellum packager builds
the standard 1×/2× ICNS representations before signing. It needs no drawing or
image dependencies at package time. To regenerate the PNG with Sharp:

```sh
node -e 'require(process.env.ELAB_SHARP_MODULE || "sharp")("assets/app-icon.svg").png().toFile("assets/app-icon.png")'
```

The recorded export uses Sharp 0.35.5. Preserve the canvas size, transparent
margin and sRGB colors when exporting with another SVG renderer.
