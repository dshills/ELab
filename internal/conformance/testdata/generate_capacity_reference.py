#!/usr/bin/env python3
"""Independent Decimal spectral RC reference; writes a new file only."""
import argparse
from decimal import Decimal as D, getcontext
import json


def eigensystem(matrix):
    """Jacobi rotations; validate orthogonality and the original eigenproblem."""
    size = len(matrix)
    original = [row[:] for row in matrix]
    vectors = [[D(i == j) for j in range(size)] for i in range(size)]
    for _ in range(200):
        p, q = max(((i, j) for i in range(size) for j in range(i+1, size)),
                   key=lambda pair: abs(matrix[pair[0]][pair[1]]))
        if abs(matrix[p][q]) < D('1e-70'):
            break
        tau = (matrix[q][q]-matrix[p][p])/(2*matrix[p][q])
        tangent = (1 if tau >= 0 else -1)/(abs(tau)+(1+tau*tau).sqrt())
        cosine = 1/(1+tangent*tangent).sqrt()
        sine = tangent*cosine
        off = matrix[p][q]
        matrix[p][p] -= tangent*off
        matrix[q][q] += tangent*off
        matrix[p][q] = matrix[q][p] = D(0)
        for i in range(size):
            if i != p and i != q:
                one, two = matrix[i][p], matrix[i][q]
                matrix[i][p] = matrix[p][i] = cosine*one-sine*two
                matrix[i][q] = matrix[q][i] = sine*one+cosine*two
            one, two = vectors[i][p], vectors[i][q]
            vectors[i][p], vectors[i][q] = cosine*one-sine*two, sine*one+cosine*two
    else:
        raise ArithmeticError('Jacobi eigensystem did not converge')
    values = [matrix[i][i] for i in range(size)]
    for i in range(size):
        for j in range(size):
            assert abs(sum(vectors[k][i]*vectors[k][j] for k in range(size))-D(i == j)) < D('1e-65')
            assert abs(sum(original[i][k]*vectors[k][j] for k in range(size))-vectors[i][j]*values[j]) < D('1e-60')
    return values, vectors


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    getcontext().prec = 80
    resistances = list(map(D, ['14000', '14000', '14000', '13000']))
    capacitance = D('1e-7')
    matrix = [[D(0)]*4 for _ in range(4)]
    for i, resistance in enumerate(resistances):
        matrix[i][i] = -1/(resistance*capacitance)
        if i:
            matrix[i][i-1] = 1/(resistance*capacitance)
        if i < 3:
            matrix[i][i] -= 1/(resistances[i+1]*capacitance)
            matrix[i][i+1] = 1/(resistances[i+1]*capacitance)
    assert all(matrix[i][j] == matrix[j][i] for i in range(4) for j in range(4))
    eigenvalues, eigenvectors = eigensystem(matrix)
    assert all(value < 0 for value in eigenvalues)

    def flow(initial, elapsed, source):
        if elapsed == 0:
            return initial[:]
        coefficients = [sum(eigenvectors[i][j]*(initial[i]-source) for i in range(4)) for j in range(4)]
        decay = [(value*elapsed).exp() for value in eigenvalues]
        return [source+sum(eigenvectors[i][j]*coefficients[j]*decay[j] for j in range(4)) for i in range(4)]

    high = D('.1')
    charged = flow([D(0)]*4, high, D(5))
    rows = []
    for text in ['0', '.000001', '.0001', '.001', '.005', '.01', '.025', '.05', '.1', '.125', '.15', '.2']:
        at = D(text)
        value = flow([D(0)]*4, at, D(5)) if at <= high else flow(charged, at-high, D(0))
        rows.append(dict(time_s=text, voltage_v=['0' if v == 0 else format(v, '.69E') for v in value]))
    result = dict(version=1, method='80-decimal-digit Jacobi eigendecomposition and scalar Decimal exponentials; checked orthogonality/eigen residuals; independent of ELab MNA, SDIRK and Go Taylor/scaling reference',
                  resistance_ohm=['14000', '14000', '14000', '13000'], capacitance_f='1e-7',
                  stimulus='0 to 5 V at startup; 5 to 0 V at 100 ms; capacitor voltage continuous at edges', rows=rows)
    with open(args.output, 'x') as output:
        output.write(json.dumps(result, indent=2)+'\n')


if __name__ == '__main__':
    main()
