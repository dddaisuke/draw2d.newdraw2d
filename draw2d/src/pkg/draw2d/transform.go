// Copyright 2010 The draw2d Authors. All rights reserved.
// created: 21/11/2010 by Laurent Le Goff

package draw2d

type MatrixTransform [6]float

const (
	epsilon = 1e-6
)

func (tr MatrixTransform) Determinant() float {
	return tr[0]*tr[3] - tr[1]*tr[2]
}

func (tr MatrixTransform) Transform(points ...*float) {
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := *points[i]
		y := *points[j]
		*points[i] = x*tr[0] + y*tr[2] + tr[4]
		*points[j] = x*tr[1] + y*tr[3] + tr[5]
	}
}

func (tr MatrixTransform) InverseTransform(points ...*float) {
	d := tr.Determinant() // matrix determinant
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := *points[i]
		y := *points[j]
		*points[i] = ((x-tr[4])*tr[3] - (y-tr[5])*tr[2]) / d
		*points[j] = ((y-tr[5])*tr[0] - (x-tr[4])*tr[1]) / d
	}
}

// ******************** Vector transformations ********************

func (tr MatrixTransform) VectorTransform(points ...*float) {
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := *points[i]
		y := *points[j]
		*points[i] = x*tr[0] + y*tr[2]
		*points[j] = y*tr[3] + x*tr[1] 
	}
}

func (tr MatrixTransform) VectorTransform2(points ...*float) {
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := *points[i]
		y := *points[j]
		*points[i] = x*tr[0] + y*tr[2]
		*points[j] = y*tr[3] + x*tr[1] 
	}
}

// ******************** Transformations creation ********************

/** Creates an identity transformation. */
func NewIdentityMatrix() MatrixTransform {
	return [6]float{1, 0, 0, 1, 0, 0}
}

/**
 * Creates a transformation with a translation, that,
 * transform point1 into point2.
 */
func NewTranslationMatrix(tx, ty float) MatrixTransform {
	return [6]float{1, 0, 0, 1, tx, ty}
}

/**
 * Creates a transformation with a sx, sy scale factor
 */
func NewScaleMatrix(sx, sy float) MatrixTransform {
	return [6]float{sx, 0, 0, sy, 0, 0}
}

/**
 * Creates a rotation transformation.
 */
func NewRotationMatrix(angle float) MatrixTransform {
	c := cos(angle)
	s := sin(angle)
	return [6]float{c, s, -s, c, 0, 0}
}

/**
 * Creates a transformation, combining a scale and a translation, that transform rectangle1 into rectangle2.
 */
func NewMatrixTransform(rectangle1, rectangle2 [4]float) MatrixTransform {
	xScale := (rectangle2[2] - rectangle2[0]) / (rectangle1[2] - rectangle1[0])
	yScale := (rectangle2[3] - rectangle2[1]) / (rectangle1[3] - rectangle1[1])
	xOffset := rectangle2[0] - (rectangle1[0] * xScale)
	yOffset := rectangle2[1] - (rectangle1[1] * yScale)
	return [6]float{xScale, 0, 0, yScale, xOffset, yOffset}
}

// ******************** Transformations operations ********************

/**
 * Returns a transformation that is the inverse of the given transformation.
 */
func (tr MatrixTransform) GetInverseTransformation() MatrixTransform {
	d := tr.Determinant() // matrix determinant
	return [6]float{
		tr[3] / d,
		-tr[1] / d,
		-tr[2] / d,
		tr[0] / d,
		(tr[2]*tr[5] - tr[3]*tr[4]) / d,
		(tr[1]*tr[4] - tr[0]*tr[5]) / d}
}


func (tr1 MatrixTransform) Multiply(tr2 MatrixTransform) MatrixTransform {
	return [6]float{
		tr1[0]*tr2[0] + tr1[1]*tr2[2],
		tr1[1]*tr2[3] + tr1[0]*tr2[1],
		tr1[2]*tr2[0] + tr1[3]*tr2[2],
		tr1[3]*tr2[3] + tr1[2]*tr2[1],
		tr1[4]*tr2[0] + tr1[5]*tr2[2] + tr2[4],
		tr1[5]*tr2[3] + tr1[4]*tr2[1] + tr2[5]}
}


func (tr *MatrixTransform) Scale(sx, sy float) (*MatrixTransform){
 	tr[0]  = tr[0]*sx;
    tr[1] = tr[1]*sx;
    tr[4]  = tr[4]*sx;
    tr[2] = tr[2]*sy;
    tr[3]  = tr[3]*sy;
    tr[5]  = tr[5]*sy;
    return tr;
}

func (tr *MatrixTransform) Translate(tx, ty float) (*MatrixTransform){
 	tr[4] = tr[4] + tx 
 	tr[5] = tr[5] + ty
    return tr;
}

func (tr *MatrixTransform) Rotate(angle float) (*MatrixTransform){
 	ca := cos(angle); 
    sa := sin(angle);
    t0 := tr[0]  * ca - tr[1] * sa;
    t2 := tr[1] * ca - tr[3] * sa;
    t4 := tr[4]  * ca - tr[5] * sa;
    tr[1] = tr[0]  * sa + tr[1] * ca;
    tr[3]  = tr[2] * sa + tr[3] * ca; 
    tr[5]  = tr[4]  * sa + tr[5] * ca;
    tr[0]  = t0;
    tr[2] = t2;
    tr[4]  = t4;
    return tr;
}

func (tr MatrixTransform) GetTranslation() (x, y float) {
	return tr[4], tr[5]
}

func (tr MatrixTransform) GetScaling() (x, y float) {
	return tr[0], tr[3]
}

func (tr MatrixTransform) GetMaxAbsScaling() (s float) {
	sx := fabs(tr[0])
	sy := fabs(tr[3])
	if(sx > sy) {
		return sx
	}
	return sy
}

func (tr MatrixTransform) GetMinAbsScaling() (s float) {
	sx := fabs(tr[0])
	sy := fabs(tr[3])
	if(sx > sy) {
		return sy
	}
	return sx
}

// ******************** Testing ********************

/**
 * Tests if a two transformation are equal. A tolerance is applied when
 * comparing matrix elements.
 */
func (tr1 MatrixTransform) Equals(tr2 MatrixTransform) bool {
	for i := 0; i < 6; i = i + 1 {
		if !fequals(tr1[i], tr2[i]) {
			return false
		}
	}
	return true
}

/**
 * Tests if a transformation is the identity transformation. A tolerance
 * is applied when comparing matrix elements.
 */
func (tr MatrixTransform) IsIdentity() bool {
	return fequals(tr[4], 0) && fequals(tr[5], 0) && tr.IsTranslation()
}

/**
 * Tests if a transformation is is a pure translation. A tolerance
 * is applied when comparing matrix elements.
 */
func (tr MatrixTransform) IsTranslation() bool {
	return fequals(tr[0], 1) && fequals(tr[1], 0) && fequals(tr[2], 0) && fequals(tr[3], 1)
}

/**
 * Compares two floats.
 * return true if the distance between the two floats is less than epsilon, false otherwise
 */
func fequals(float1, float2 float) bool {
	return fabs(float1-float2) <= epsilon
}

type VertexMatrixTransform struct {
	tr		MatrixTransform
	Next	VertexConverter
}

func NewVertexMatrixTransform(tr MatrixTransform, converter VertexConverter) (*VertexMatrixTransform){
	return &VertexMatrixTransform{tr, converter}
}

// Vertex Matrix Transform
func (vmt *VertexMatrixTransform) NextCommand(command VertexCommand) {
	vmt.Next.NextCommand(command)
}

func (vmt *VertexMatrixTransform) Vertex(x, y float) {
	vmt.tr.Transform(&x, &y)
	vmt.Next.Vertex(x, y)
}
