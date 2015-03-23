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
	if e == beBreak {
		return "break"
	}
	return "continue"
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
		switch st.Tok {
		case token.DEFINE:
			// Check to make sure at least one new variables and not constants in Lhs
			hasNew := false
			for _, l := range st.Lhs {
				ident := l.(*ast.Ident)
				v := ns.FindLocal(ident.Name)
				if v == NoValue {
					hasNew = true
				} else if v.Type() == ConstValueType {
					return cannotAssignToErr(l)
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
				v := ns.FindLocal(lIdent.Name)
				vl := values[i]
				if v == NoValue {
					vl = removeBasicLit(vl)
					v = reflect.New(vl.Type()).Elem()
					ns.AddLocal(lIdent.Name, v)
				} else {
					vl = matchDestType(vl, v.Type())
				}

				if err := assignTo(v, vl); err != nil {
					return err
				}
			}

		case token.ASSIGN:
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
				v, err := checkSingleValue(mch.evalExpr(ns, l))
				if err != nil {
					return err
				}

				if v.Type() == MapIndexValueType {
					v := v.Interface().(MapIndexValue)
					values[i] = matchDestType(values[i], v.X.Type().Elem())
					v.X.SetMapIndex(v.Key, values[i])
					continue
				}
				if !v.CanSet() {
					return cannotAssignToErr(l)
				}
				v.Set(matchDestType(values[i], v.Type()))
			}

		case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
			l := st.Lhs[0]
			v, err := checkSingleValue(mch.evalExpr(ns, l))
			if err != nil {
				return err
			}

			if !v.CanSet() {
				return cannotAssignToErr(l)
			}

			r := st.Rhs[0]
			delta, err := checkSingleValue(mch.evalExpr(ns, r))
			if err != nil {
				return err
			}
			delta = matchDestType(delta, v.Type())
			var newV interface{}
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				switch st.Tok {
				case token.ADD_ASSIGN:
					newV = v.Int() + delta.Int()
				case token.SUB_ASSIGN:
					newV = v.Int() - delta.Int()
				case token.MUL_ASSIGN:
					newV = v.Int() * delta.Int()
				case token.QUO_ASSIGN:
					newV = v.Int() / delta.Int()
				case token.REM_ASSIGN:
					newV = v.Int() % delta.Int()
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				switch st.Tok {
				case token.ADD_ASSIGN:
					newV = v.Uint() + delta.Uint()
				case token.SUB_ASSIGN:
					newV = v.Uint() - delta.Uint()
				case token.MUL_ASSIGN:
					newV = v.Uint() * delta.Uint()
				case token.QUO_ASSIGN:
					newV = v.Uint() / delta.Uint()
				case token.REM_ASSIGN:
					newV = v.Uint() % delta.Uint()
				}
			case reflect.Float32, reflect.Float64:
				switch st.Tok {
				case token.ADD_ASSIGN:
					newV = v.Float() + delta.Float()
				case token.SUB_ASSIGN:
					newV = v.Float() - delta.Float()
				case token.MUL_ASSIGN:
					newV = v.Float() * delta.Float()
				case token.QUO_ASSIGN:
					newV = v.Float() / delta.Float()
				case token.REM_ASSIGN:
					return invalidOperationErr(st.Tok.String(), v.Type())
				}
			case reflect.Complex64, reflect.Complex128:
				switch st.Tok {
				case token.ADD_ASSIGN:
					newV = v.Complex() + delta.Complex()
				case token.SUB_ASSIGN:
					newV = v.Complex() - delta.Complex()
				case token.MUL_ASSIGN:
					newV = v.Complex() * delta.Complex()
				case token.QUO_ASSIGN:
					newV = v.Complex() / delta.Complex()
				case token.REM_ASSIGN:
					return invalidOperationErr(st.Tok.String(), v.Type())
				}
			case reflect.String:
				switch st.Tok {
				case token.ADD_ASSIGN:
					newV = v.String() + delta.String()
				default:
					return invalidOperationErr(st.Tok.String(), v.Type())
				}
			default:
				return invalidOperationErr(st.Tok.String(), v.Type())
			}
			v.Set(reflect.ValueOf(newV).Convert(v.Type()))
		}
		return nil

	case *ast.ExprStmt:
		_, err := mch.evalExpr(ns, st.X)
		return err

	case *ast.DeclStmt:
		switch decl := st.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				isConst := decl.Tok == token.CONST
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
					if ns.FindLocal(name.Name) != NoValue {
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
						if tp, err = mch.evalType(ns, spec.Type); err != nil {
							return err
						}
						if values != nil {
							vl = matchDestType(vl, tp)
						}
					} else {
						if !isConst {
							// a variable cannot take basic lit types.
							vl = removeBasicLit(vl)
						}
						tp = vl.Type()
					}
					pv = reflect.New(tp)

					if values != nil {
						pv.Elem().Set(vl)
					}
					if isConst {
						ns.AddLocal(name.Name, ToConstant(pv.Elem()))
					} else {
						ns.AddLocal(name.Name, pv.Elem())
					}
				}
			}
			return nil

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
		return nil

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
				if brk, ok := err.(BranchErr); ok {
					if brk == beBreak {
						break
					}
				} else {
					return err
				}
			}

			if st.Post != nil {
				if err := mch.runStatement(blkNs, st.Post); err != nil {
					return err
				}
			}
		}
		return nil

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
			if st.Else != nil {
				if err := mch.runStatement(blkNs, st.Else); err != nil {
					return err
				}
			}
		}
		return nil

	case *ast.IncDecStmt:
		x, err := checkSingleValue(mch.evalExpr(ns, st.X))
		if err != nil {
			return err
		}

		if !x.CanSet() {
			return cannotAssignToErr(st.X)
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

	case *ast.SwitchStmt:
		blkNs := ns
		if st.Init != nil {
			blkNs = ns.NewBlock()
			mch.runStatement(blkNs, st.Init)
		}

		tag := trueValue
		if st.Tag != nil {
			var err error
			if tag, err = checkSingleValue(mch.evalExpr(blkNs, st.Tag)); err != nil {
				return err
			}
		}

		for _, el := range st.Body.List {
			cc := el.(*ast.CaseClause)
			matched := len(cc.List) == 0
			for _, el := range cc.List {
				vl, err := checkSingleValue(mch.evalExpr(blkNs, el))
				if err != nil {
					return err
				}

				tag := tag
				if tag, vl, err = matchType(tag, vl); err != nil {
					return err
				}

				eq, err := valueEqual(tag, vl)
				if err != nil {
					return err
				}
				if eq {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}

			caseBlkNs := blkNs.NewBlock()
			for _, bodySt := range cc.Body {
				err := mch.runStatement(caseBlkNs, bodySt)
				if err != nil {
					// TODO support fallthrough, break
					return err
				}
			}
			break
		}

		return nil
	}

	log.Println("Unknown statement type")
	ast.Print(token.NewFileSet(), st)
	panic("")
	return fmt.Errorf("Unknown statement type")
}
