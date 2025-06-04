package compat

import gosat "github.com/crillab/gophersat/solver"

func ConvertEbuildToSat(ebuildPath string) ([]gosat.Clause, error) {
	metadata := parseEbuild(ebuildPath)
	var clauses []gosat.Clause

	// Ограничение версии
	clauses = append(clauses, gosat.AtLeast1(
		genVersionVar(metadata.Name, metadata.Version),
	))

	// USE-флаги
	for flag, enabled := range metadata.UseFlags {
		clause := gosat.NewClause()
		if enabled {
			clause.AddLit(genUseFlagVar(metadata.Name, flag))
		} else {
			clause.AddLit(-genUseFlagVar(metadata.Name, flag))
		}
		clauses = append(clauses, clause)
	}
	return clauses, nil
}
