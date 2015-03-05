package gsvm

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"reflect"
)

type BranchErr int

const (
	beBreak BranchErr = iota
	beContinue
)

func (e BranchErr) Error() string {
	return ""
}

func assignTo(v reflect.Value, vl reflect.Value) error {
	if v.Type() != vl.Type() {
		return cannotUseAsInAssignmentErr(vl, v.Type())
	}
	v.Set(vl)
	return nil
}

func (mch *machine) runStatement(ns NameSpace, st ast.Stmt) error {
	switch st := st.(type) {
	case *ast.AssignStmt:
		// TODO when len(st.Rhs) == 1, check multi value return
		if len(st.Lhs) != len(st.Rhs) {
			return fmt.Errorf("assignment count mismatch: %d %s %d", len(st.Lhs), st.Tok, len(st.Rhs))
		}
		if st.Tok == token.DEFINE {
			// Check to make sure at least one new variables
			hasNew := false
			for _, l := range st.Lhs {
				lIdent := l.(*ast.Ident)
				if ns.FindLocalVar(lIdent.Name) == noValue {
					hasNew = true
					break
				}
			}
			if !hasNew {
				return noNewVarsErr
			}

			// Compute values
			values := make([]reflect.Value, len(st.Rhs))
			for i, r := range st.Rhs {
				rV, err := checkSingleValue(mch.evalExpr(ns, r))
				if err != nil {
					return err
				}
				if len(st.Rhs) > 1 && rV.CanAddr() {
					// Make a copy of lvalue for parallel assignments
					tmp := reflect.New(rV.Type())
					tmp.Elem().Set(rV)
					rV = tmp.Elem()
				}
				values[i] = rV
			}

			// Define and assign
			for i, l := range st.Lhs {
				lIdent := l.(*ast.Ident)
				pv := ns.FindLocalVar(lIdent.Name)
				vl := values[i]
				if pv == noValue {
					vl = removeBasicLit(vl)
					pv = reflect.New(vl.Type())
					ns.AddLocalVar(lIdent.Name, pv)
				} else {
					vl = matchDestType(vl, pv.Elem().Type())
				}

				if err := assignTo(pv.Elem(), vl); err != nil {
					return err
				}
			}
		} else {
			values := make([]reflect.Value, len(st.Rhs))
			for i, r := range st.Rhs {
				rV, err := checkSingleValue(mch.evalExpr(ns, r))
				if err != nil {
					return err
				}
				if len(st.Rhs) > 1 && rV.CanAddr() {
					// Make a copy of lvalue for parallel assignments
					tmp := reflect.New(rV.Type())
					tmp.Elem().Set(rV)
					rV = tmp.Elem()
				}
				values[i] = rV
			}
			for i, l := range st.Lhs {
				lV, err := checkSingleValue(mch.evalExpr(ns, l))
				if err != nil {
					return err
				}
				if !lV.CanSet() {
					return cannotAssignToErr(lV.String())
				}
				lV.Set(values[i])
			}
		}

	case *ast.ExprStmt:
		_, err := mch.evalExpr(ns, st.X)
		return err

	case *ast.DeclStmt:
		switch decl := st.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				spec := spec.(*ast.ValueSpec)
				var values []reflect.Value
				if len(spec.Values) == 1 {
					var err error
					if values, err = mch.evalExpr(ns, spec.Values[0]); err != nil {
						return err
					}
				} else if len(spec.Values) > 1 {
					values = make([]reflect.Value, len(spec.Values))
					for i, valueExpr := range spec.Values {
						value, err := checkSingleValue(mch.evalExpr(ns, valueExpr))
						if err != nil {
							return err
						}
						values[i] = value
					}
				} else if spec.Type == nil {
					return fmt.Errorf("Need type")
				}

				if values != nil && len(spec.Names) != len(values) {
					return fmt.Errorf("assignment count mismatch: %d = %d", len(spec.Names), len(values))
				}

				for i, name := range spec.Names {
					if ns.FindLocalVar(name.Name) != noValue {
						return redeclareVarErr(name.Name)
					}
					var pv reflect.Value
					var tp reflect.Type
					var vl reflect.Value
					if values != nil {
						vl = values[i]
					}
					if spec.Type != nil {
						var err error
						if tp, err = mch.evalType(spec.Type); err != nil {
							return err
						}
						if values != nil {
							vl = matchDestType(vl, tp)
						}
					} else {
						vl = removeBasicLit(vl)
						tp = vl.Type()
					}
					pv = reflect.New(tp)

					if values != nil {
						pv.Elem().Set(vl)
					}
					ns.AddLocalVar(name.Name, pv)
				}
			}

		case *ast.FuncDecl:
			// TODO
			ast.Print(token.NewFileSet(), decl)
			return nil
		}

	case *ast.BlockStmt:
		blkNs := ns.NewBlock()
		for _, st := range st.List {
			if err := mch.runStatement(blkNs, st); err != nil {
				return err
			}
		}

	case *ast.ForStmt:
		blkNs := ns
		if st.Init != nil {
			blkNs = ns.NewBlock()
			mch.runStatement(blkNs, st.Init)
		}

		for {
			cond := true
			if st.Cond != nil {
				cnd, err := checkSingleValue(mch.evalExpr(blkNs, st.Cond))
				if err != nil {
					return err
				}

				if cnd.Kind() != reflect.Bool {
					return nonBoolAsConditionErr(cnd, "for")
				}
				cond = cnd.Bool()
			}
			if !cond {
				break
			}

			if err := mch.runStatement(blkNs, st.Body); err != nil {
				return err
			}

			if st.Post != nil {
				if err := mch.runStatement(blkNs, st.Post); err != nil {
					return err
				}
			}
		}

	case *ast.BranchStmt:
		if st.Tok == token.BREAK {
			return beBreak
		} else {
			return beContinue
		}

	case *ast.IfStmt:
		blkNs := ns
		if st.Init != nil {
			blkNs = ns.NewBlock()
			mch.runStatement(blkNs, st.Init)
		}

		cnd, err := checkSingleValue(mch.evalExpr(blkNs, st.Cond))
		if err != nil {
			return err
		}

		if cnd.Kind() != reflect.Bool {
			return nonBoolAsConditionErr(cnd, "for")
		}

		if cnd.Bool() {
			if err := mch.runStatement(blkNs, st.Body); err != nil {
				return err
			}
		} else {
			if err := mch.runStatement(blkNs, st.Else); err != nil {
				return err
			}
		}

	case *ast.IncDecStmt:
		x, err := checkSingleValue(mch.evalExpr(ns, st.X))
		if err != nil {
			return err
		}

		if !x.CanSet() {
			return cannotAssignToErr(x.String())
		}

		switch x.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if st.Tok == token.INC {
				x.SetInt(x.Int() + 1)
			} else {
				x.SetInt(x.Int() - 1)
			}
			return nil

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if st.Tok == token.INC {
				x.SetUint(x.Uint() + 1)
			} else {
				x.SetUint(x.Uint() - 1)
			}
			return nil

		case reflect.Float32, reflect.Float64:
			if st.Tok == token.INC {
				x.SetFloat(x.Float() + 1)
			} else {
				x.SetFloat(x.Float() - 1)
			}
			return nil

		case reflect.Complex64, reflect.Complex128:
			if st.Tok == token.INC {
				x.SetComplex(x.Complex() + 1)
			} else {
				x.SetComplex(x.Complex() - 1)
			}
			return nil

		default:
			return invalidOperationErr(st.Tok.String(), x.Type())
		}
		panic("")

	default:
		log.Println("Unknown statement type")
		ast.Print(token.NewFileSet(), st)
		return fmt.Errorf("Unknown statement type")
	}
	return nil
}
