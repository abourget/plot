// The veceps implemens the vecgfx.Canvas interface using
// encapsulated postscript.
package veceps

import (
	"bufio"
	"bytes"
	"code.google.com/p/plotinum/vecgfx"
	"fmt"
	"image/color"
	"math"
	"os"
	"time"
)

type EpsCanvas struct {
	stk []ctx
	buf *bytes.Buffer
}

type ctx struct {
	color  color.Color
	width  float64
	dashes []float64
	offs   float64
	font   string
	fsize  float64
}

// pr is the amount of precision to use when outputting float64s.
const pr = 5

// New returns a new EpsCanvas with the width and height given
// in inches and a title string.
func New(w, h float64, title string) *EpsCanvas {
	s := "%%!PS-Adobe-3.0 EPSF-3.0\n"
	s += "%%Creator code.google.com/p/plotinum/vecgfx/veceps\n"
	s += "%%Title: " + title + "\n"
	s += fmt.Sprintf("%%%%BoundingBox 0 0 %.*g %.*g\n", pr, w*vecgfx.PtInch,
		pr, h*vecgfx.PtInch)
	s += fmt.Sprintf("%%%%CreationDate: %s\n", time.Now())
	s += "%%Orientation: Portrait\n"
	s += "%%EndComments\n"
	s += "\n"
	s += "0 0 0 setrgbcolor\n"
	s += "1 setlinewidth\n"
	s += "[] 0 setdash\n"
	c := ctx{
		color:  color.RGBA{A: 255},
		width:  1,
		dashes: []float64{},
		offs:   0,
	}
	return &EpsCanvas{
		stk: []ctx{c},
		buf: bytes.NewBufferString(s),
	}
}

// cur returns the top context on the stack.
func (e *EpsCanvas) cur() *ctx {
	return &e.stk[len(e.stk)-1]
}

func (e *EpsCanvas) SetLineWidth(w float64) {
	if e.cur().width != w {
		e.cur().width = w
		fmt.Fprintf(e.buf, "%.*g setlinewidth\n", pr, w)
	}
}

func (e *EpsCanvas) SetLineDash(dashes []float64, o float64) {
	dashEq := true
	curDash := e.cur().dashes
	for i, d := range dashes {
		if d != curDash[i] {
			dashEq = false
			break
		}
	}
	if !dashEq || e.cur().offs != o {
		e.cur().dashes = dashes
		e.cur().offs = o
		e.buf.WriteString("[")
		for _, d := range dashes {
			fmt.Fprintf(e.buf, " %.*g", pr, d)
		}
		e.buf.WriteString(" ] ")
		fmt.Fprintf(e.buf, "%.*g setdash\n", pr, o)
	}
}

func (e *EpsCanvas) SetColor(c color.Color) {
	if e.cur().color != c {
		e.cur().color = c
		r, g, b, _ := c.RGBA()
		mx := float64(math.MaxUint16)
		fmt.Fprintf(e.buf, "%.*g %.*g %.*g setrgbcolor\n", pr, float64(r)/mx,
			pr, float64(g)/mx, pr, float64(b)/mx)
	}
}

func (e *EpsCanvas) Rotate(r float64) {
	fmt.Fprintf(e.buf, "%.*g rotate\n", pr, r*180/math.Pi)
}

func (e *EpsCanvas) Translate(x float64, y float64) {
	fmt.Fprintf(e.buf, "%.*g %.*g translate\n", pr, x, pr, y)
}

func (e *EpsCanvas) Scale(x float64, y float64) {
	fmt.Fprintf(e.buf, "%.*g %.*g scale\n", pr, x, pr, y)
}

func (e *EpsCanvas) Push() {
	e.stk = append(e.stk, *e.cur())
	e.buf.WriteString("gsave\n")
}

func (e *EpsCanvas) Pop() {
	e.stk = e.stk[:len(e.stk)-1]
	e.buf.WriteString("grestore\n")
}

func (e *EpsCanvas) Stroke(path vecgfx.Path) {
	e.trace(path)
	e.buf.WriteString("stroke\n")
}

func (e *EpsCanvas) Fill(path vecgfx.Path) {
	e.trace(path)
	e.buf.WriteString("fill\n")
}

func (e *EpsCanvas) trace(path vecgfx.Path) {
	e.buf.WriteString("newpath\n")
	for _, comp := range path {
		switch comp.Type {
		case vecgfx.MoveComp:
			fmt.Fprintf(e.buf, "%.*g %.*g moveto\n", pr, comp.X, pr, comp.Y)
		case vecgfx.LineComp:
			fmt.Fprintf(e.buf, "%.*g %.*g lineto\n", pr, comp.X, pr, comp.Y)
		case vecgfx.ArcComp:
			fmt.Fprintf(e.buf, "%.*g %.*g %.*g %.*g %.*g arc\n", pr, comp.X, pr, comp.Y,
				pr, comp.Radius, pr, comp.Start*180/math.Pi, pr,
				comp.Finish*180/math.Pi)
		case vecgfx.CloseComp:
			e.buf.WriteString("closepath\n")
		default:
			panic(fmt.Sprintf("Unknown path component type: %d\n", comp.Type))
		}
	}
}

func (e *EpsCanvas) FillText(fnt vecgfx.Font, x float64, y float64, str string) {
	if e.cur().font != fnt.Name() || e.cur().fsize != fnt.Size {
		e.cur().font = fnt.Name()
		e.cur().fsize = fnt.Size
		fmt.Fprintf(e.buf, "/%s findfont %.*g scalefont setfont\n",
			fnt.Name(), pr, fnt.Size)
	}
	fmt.Fprintf(e.buf, "%.*g %.*g moveto\n", pr, x, pr, y)
	fmt.Fprintf(e.buf, "(%s) show\n", str)
}

func (e *EpsCanvas) DPI() float64 {
	return vecgfx.PtInch
}

// Save saves the plot to the given path.
func (e *EpsCanvas) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	b := bufio.NewWriter(f)
	_, err = e.buf.WriteTo(b)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(b, "showpage")
	if err != nil {
		return err
	}
	return b.Flush()
}
